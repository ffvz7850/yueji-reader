package store

import (
	"strings"
	"testing"
)

func setupFolderTestData(t *testing.T) {
	t.Helper()
	setupTestDB(t)

	if _, err := db.Exec(`
		INSERT INTO "Library" ("id", "name", "rootPath", "type", "enabled") VALUES
			('folder-lib', '写真库', '/data/photos', 'comic', 1)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO "Comic" ("id", "filename", "title", "type", "libraryId", "relativePath", "addedAt") VALUES
			('f-root-a',    'A套.cbz',          'A套',          'comic', 'folder-lib', 'A套.cbz',          '2026-01-01 00:00:00'),
			('f-root-dir1', '摄影师1/',           '摄影师1',       'comic', 'folder-lib', '摄影师1/',          '2026-01-01 00:00:00'),
			('f-dir1-a',    '摄影师1/套装1/',      '套装1',         'comic', 'folder-lib', '摄影师1/套装1/',    '2026-01-02 00:00:00'),
			('f-dir1-b',    '摄影师1/套装2.cbz',  '套装2',         'comic', 'folder-lib', '摄影师1/套装2.cbz', '2026-01-03 00:00:00'),
			('f-dir1-c',    '摄影师1/子目录/散集/', '散集',         'comic', 'folder-lib', '摄影师1/子目录/散集/', '2026-01-04 00:00:00'),
			('f-root-dir2', '摄影师2/',           '摄影师2',       'comic', 'folder-lib', '摄影师2/',          '2026-01-05 00:00:00'),
			('f-dir2-a',    '摄影师2/胶片/',       '胶片',         'comic', 'folder-lib', '摄影师2/胶片/',      '2026-01-06 00:00:00'),
			('f-novel',     'book.epub',         '小说',         'novel', 'folder-lib', 'book.epub',        '2026-01-07 00:00:00'),
			('f-missing',   '摄影师1/已删/',       '已删',         'comic', 'folder-lib', '摄影师1/已删/',      '2026-01-08 00:00:00')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE "Comic" SET "missingSince" = '2026-02-01 00:00:00' WHERE "id" = 'f-missing'`); err != nil {
		t.Fatal(err)
	}
}

func TestGetLibraryFolderTreeRoot(t *testing.T) {
	setupFolderTestData(t)

	res, err := GetLibraryFolderTree("folder-lib", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.LibraryName != "写真库" {
		t.Errorf("LibraryName = %q, want 写真库", res.LibraryName)
	}
	if res.CurrentPath != "" || res.ParentPath != "" || len(res.Segments) != 0 {
		t.Errorf("root meta mismatch: %+v", res)
	}
	// 根目录直接作品：A套.cbz + 摄影师1/ + 摄影师2/（图片文件夹直接位于根时既是文件夹也是作品；novel 与缺失项不计入）
	if res.TotalAlbums != 3 {
		t.Errorf("TotalAlbums = %d, want 3", res.TotalAlbums)
	}
	// 直接子文件夹：摄影师1、摄影师2
	if len(res.Folders) != 2 {
		t.Fatalf("folders = %d, want 2: %+v", len(res.Folders), res.Folders)
	}
	f1, f2 := res.Folders[0], res.Folders[1]
	if f1.Name != "摄影师1" || f2.Name != "摄影师2" {
		t.Errorf("folder names = %q,%q", f1.Name, f2.Name)
	}
	if f1.Path != "摄影师1" || f2.Path != "摄影师2" {
		t.Errorf("folder paths = %q,%q", f1.Path, f2.Path)
	}
	// 摄影师1 子树：套装1、套装2、子目录/散集（3 本，缺失的已删不计）
	if f1.AlbumCount != 3 {
		t.Errorf("摄影师1 AlbumCount = %d, want 3", f1.AlbumCount)
	}
	// 摄影师1 直接子项：套装1、套装2、子目录（3 项）
	if f1.ItemCount != 3 {
		t.Errorf("摄影师1 ItemCount = %d, want 3", f1.ItemCount)
	}
	// 封面取子树最早入库：套装1
	if !strings.Contains(f1.CoverURL, "f-dir1-a") {
		t.Errorf("摄影师1 CoverURL = %q, want contains f-dir1-a", f1.CoverURL)
	}
	// 摄影师2：1 本
	if f2.AlbumCount != 1 || f2.ItemCount != 1 {
		t.Errorf("摄影师2 counts = %d/%d, want 1/1", f2.AlbumCount, f2.ItemCount)
	}
}

func TestGetLibraryFolderTreeSubdir(t *testing.T) {
	setupFolderTestData(t)

	res, err := GetLibraryFolderTree("folder-lib", "摄影师1")
	if err != nil {
		t.Fatal(err)
	}
	if res.CurrentPath != "摄影师1" || res.ParentPath != "" || len(res.Segments) != 1 || res.Segments[0] != "摄影师1" {
		t.Errorf("meta mismatch: current=%q parent=%q segments=%v", res.CurrentPath, res.ParentPath, res.Segments)
	}
	// 摄影师1 直接作品：套装1（图片文件夹）、套装2（归档）
	if res.TotalAlbums != 2 {
		t.Errorf("TotalAlbums = %d, want 2", res.TotalAlbums)
	}
	// 直接子文件夹：子目录
	if len(res.Folders) != 1 || res.Folders[0].Name != "子目录" {
		t.Fatalf("folders = %+v, want [子目录]", res.Folders)
	}
	if res.Folders[0].Path != "摄影师1/子目录" {
		t.Errorf("folder path = %q", res.Folders[0].Path)
	}
	if res.Folders[0].AlbumCount != 1 || res.Folders[0].ItemCount != 1 {
		t.Errorf("子目录 counts = %d/%d, want 1/1", res.Folders[0].AlbumCount, res.Folders[0].ItemCount)
	}
	if !strings.Contains(res.Folders[0].CoverURL, "f-dir1-c") {
		t.Errorf("子目录 CoverURL = %q, want contains f-dir1-c", res.Folders[0].CoverURL)
	}
}

func TestGetAllComicsFolderFilter(t *testing.T) {
	setupFolderTestData(t)

	// 根目录直接子项：A套.cbz + 摄影师1/ + 摄影师2/（图片文件夹直接位于根时既是文件夹也是作品）
	// novel 不计入（ContentType=comic）
	res, err := GetAllComics(ComicListOptions{
		ContentType:      "comic",
		FilterLibraryIDs: true,
		LibraryIDs:       []string{"folder-lib"},
		Folder:           "",
		FolderFilter:     true,
		Page:             1,
		PageSize:         50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 3 {
		t.Errorf("root direct children = %d, want 3: %+v", res.Total, res.Comics)
	}
	gotRoot := map[string]bool{}
	for _, c := range res.Comics {
		gotRoot[c.ID] = true
	}
	if !gotRoot["f-root-a"] || !gotRoot["f-root-dir1"] || !gotRoot["f-root-dir2"] {
		t.Errorf("unexpected root children: %v", gotRoot)
	}

	// 摄影师1 目录直接子项：套装1、套装2（子目录/散集 不计入）
	// 缺失项（已删）与平铺视图行为一致：仍会列出
	res2, err := GetAllComics(ComicListOptions{
		ContentType:      "comic",
		FilterLibraryIDs: true,
		LibraryIDs:       []string{"folder-lib"},
		Folder:           "摄影师1",
		FolderFilter:     true,
		Page:             1,
		PageSize:         50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != 3 {
		t.Fatalf("摄影师1 direct children = %d, want 3: %+v", res2.Total, res2.Comics)
	}
	got := map[string]bool{}
	for _, c := range res2.Comics {
		got[c.ID] = true
	}
	if !got["f-dir1-a"] || !got["f-dir1-b"] || !got["f-missing"] {
		t.Errorf("unexpected children: %v", got)
	}

	// 子目录：散集（图片文件夹带尾部斜杠也要命中）
	res3, err := GetAllComics(ComicListOptions{
		ContentType:      "comic",
		FilterLibraryIDs: true,
		LibraryIDs:       []string{"folder-lib"},
		Folder:           "摄影师1/子目录",
		FolderFilter:     true,
		Page:             1,
		PageSize:         50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res3.Total != 1 || res3.Comics[0].ID != "f-dir1-c" {
		t.Errorf("子目录 direct children = %+v, want f-dir1-c", res3.Comics)
	}
}
