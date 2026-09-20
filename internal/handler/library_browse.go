package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nowen-reader/nowen-reader/internal/middleware"
	"github.com/nowen-reader/nowen-reader/internal/store"
)

// LibraryBrowseHandler 处理"文件夹式浏览"请求：把图库按目录组织成树，
// 一次返回当前目录下的子文件夹（带封面缩略图）与统计信息。
// 当前目录下的直接作品列表复用 GET /api/comics?folder=... 返回。
type LibraryBrowseHandler struct{}

// NewLibraryBrowseHandler 创建 LibraryBrowseHandler。
func NewLibraryBrowseHandler() *LibraryBrowseHandler {
	return &LibraryBrowseHandler{}
}

// GET /api/library-folders?libraryId=xxx&path=sub%2Ffolder
// 返回指定书库、指定相对目录（path="" 表示库根）下的子文件夹列表。
func (h *LibraryBrowseHandler) BrowseFolders(c *gin.Context) {
	libraryID := strings.TrimSpace(c.Query("libraryId"))
	folder := store.NormalizeFolderPath(c.Query("path"))
	if libraryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	// 权限校验：管理员任意访问；普通用户仅限可见书库（与 /api/comics 一致）。
	uid := getUserID(c)
	if uid != "" {
		user := middleware.GetCurrentUser(c)
		allowed := false
		if user != nil && user.Role == "admin" {
			allowed = true
		} else if ok, err := store.UserCanViewLibrary(uid, libraryID); err == nil && ok {
			allowed = true
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此书库"})
			return
		}
	}

	result, err := store.GetLibraryFolderTree(libraryID, folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to browse library folders"})
		return
	}

	c.Header("Cache-Control", "private, max-age=15, stale-while-revalidate=60")
	c.JSON(http.StatusOK, result)
}
