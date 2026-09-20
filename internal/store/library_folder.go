package store

import (
	"sort"
	"strings"
	"time"
)

// ============================================================
// 文件夹式浏览（Folder Browsing）
// ============================================================
// 图库中的作品（含"图片文件夹漫画"，如 写真集/套装A/）都带相对库根的
// relativePath（正斜杠，图片文件夹以 "/" 结尾）。文件夹视图按该路径把作品
// 组织成树：一次返回当前目录的直接子文件夹（带封面缩略图）与统计信息。
// 某个目录下的直接作品列表由 /api/comics?folder=... 复用现有查询返回。

// LibraryFolderEntry 表示当前目录下的一个子文件夹。
type LibraryFolderEntry struct {
	Name       string `json:"name"`       // 文件夹名
	Path       string `json:"path"`       // 相对库根的完整路径
	CoverURL   string `json:"coverUrl"`   // 封面缩略图（取该文件夹内最早入库作品的封面）
	AlbumCount int    `json:"albumCount"` // 文件夹内（含子文件夹）作品总数
	ItemCount  int    `json:"itemCount"`  // 直接子项数（子文件夹 + 直接作品）
}

// LibraryBrowseResult 是文件夹浏览接口的返回结构。
type LibraryBrowseResult struct {
	LibraryID    string               `json:"libraryId"`
	LibraryName  string               `json:"libraryName"`
	CurrentPath  string               `json:"currentPath"` // 相对库根，"" = 根
	ParentPath   string               `json:"parentPath"`  // 父目录，"" = 无
	Segments     []string             `json:"segments"`    // 面包屑路径分段
	Folders      []LibraryFolderEntry `json:"folders"`
	TotalAlbums  int                  `json:"totalAlbums"`  // 当前目录直接作品数
	TotalFolders int                  `json:"totalFolders"` // 当前目录直接子文件夹数
}

// NormalizeFolderPath 规范化相对目录：统一正斜杠、去首尾斜杠、压缩重复斜杠。
func NormalizeFolderPath(folder string) string {
	folder = strings.ReplaceAll(folder, "\\", "/")
	folder = strings.Trim(folder, "/")
	for strings.Contains(folder, "//") {
		folder = strings.ReplaceAll(folder, "//", "/")
	}
	return folder
}

type folderAgg struct {
	name       string
	coverID    string
	coverTime  time.Time
	albumCount int
	childSet   map[string]struct{}
}

// GetLibraryFolderTree 返回指定书库、指定相对目录下的文件夹列表与统计。
// 覆盖规则：
//   - 只统计 type='comic' 且未丢失的作品（novel 走独立书架，不参与文件夹视图）；
//   - 子文件夹由作品路径推导：目录下所有作品路径的首段即直接子文件夹；
//   - 文件夹封面取该文件夹子树中最早入库（addedAt，并列取 id）作品的缩略图；
//   - ItemCount 为该文件夹的直接子项（直接作品 + 直接子文件夹）去重后的数量。
func GetLibraryFolderTree(libraryID, folder string) (*LibraryBrowseResult, error) {
	result := &LibraryBrowseResult{
		LibraryID:   libraryID,
		CurrentPath: NormalizeFolderPath(folder),
		Folders:     []LibraryFolderEntry{},
	}
	result.Segments = splitSegments(result.CurrentPath)
	if result.CurrentPath != "" {
		result.ParentPath = parentOf(result.CurrentPath)
	}

	if lib, err := GetLibraryByID(libraryID); err == nil && lib != nil {
		result.LibraryName = lib.Name
	}

	// 只拉取必要的列，按库索引扫描；非根目录时用前缀过滤减少数据量。
	query := `SELECT "id", "relativePath", "addedAt" FROM "Comic" WHERE "libraryId" = ? AND "type" = 'comic' AND "missingSince" IS NULL AND "relativePath" != ''`
	args := []interface{}{libraryID}
	if result.CurrentPath != "" {
		query += ` AND "relativePath" LIKE ? ESCAPE '\'`
		args = append(args, escapeLikePattern(result.CurrentPath+"/")+"%")
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefix := ""
	if result.CurrentPath != "" {
		prefix = result.CurrentPath + "/"
	}

	folders := make(map[string]*folderAgg)
	for rows.Next() {
		var id, rel string
		var addedAt time.Time
		if err := rows.Scan(&id, &rel, &addedAt); err != nil {
			return nil, err
		}
		p := strings.TrimSuffix(rel, "/")
		if p == "" {
			continue
		}
		var rest string
		if prefix != "" {
			if !strings.HasPrefix(p, prefix) {
				continue
			}
			rest = p[len(prefix):]
		} else {
			rest = p
		}
		if rest == "" {
			continue // 目录本身不构成作品
		}
		parts := strings.Split(rest, "/")
		if len(parts) == 1 {
			result.TotalAlbums++ // 当前目录的直接作品
			continue
		}
		seg := parts[0]
		agg, ok := folders[seg]
		if !ok {
			agg = &folderAgg{name: seg, childSet: make(map[string]struct{})}
			folders[seg] = agg
		}
		agg.albumCount++
		if agg.coverID == "" || addedAt.Before(agg.coverTime) || (addedAt.Equal(agg.coverTime) && id < agg.coverID) {
			agg.coverID = id
			agg.coverTime = addedAt
		}
		if len(parts) >= 2 {
			agg.childSet[parts[1]] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(folders))
	for name := range folders {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	for _, name := range names {
		agg := folders[name]
		folderPath := name
		if result.CurrentPath != "" {
			folderPath = result.CurrentPath + "/" + name
		}
		result.Folders = append(result.Folders, LibraryFolderEntry{
			Name:       name,
			Path:       folderPath,
			CoverURL:   BuildComicCoverURL(agg.coverID),
			AlbumCount: agg.albumCount,
			ItemCount:  len(agg.childSet),
		})
	}
	result.TotalFolders = len(result.Folders)
	return result, nil
}

func splitSegments(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func parentOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// escapeLikePattern 转义 LIKE 模式中的 % 与 _，配合 ESCAPE '\' 使用。
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
