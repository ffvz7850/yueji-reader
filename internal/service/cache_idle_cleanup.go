package service

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nowen-reader/nowen-reader/internal/config"
)

// StartCacheIdleCleanup 阅读页缓存空闲清理：
// 无活跃阅读会话、且页面缓存目录超过 30 分钟无任何写入时，
// 清空整个页面缓存目录——即"退出浏览器后约 30 分钟自动清理干净"，
// 避免浏览过的图长期占用磁盘。缩略图缓存不受影响。
func StartCacheIdleCleanup() {
	go func() {
		time.Sleep(30 * time.Second) // 给启动扫描/首次 warmup 留时间
		cleanupIdlePageCache()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupIdlePageCache()
		}
	}()
	log.Println("[cache-idle] Page cache idle cleanup started (idle threshold: 30m, check interval: 5m)")
}

// idleCacheThreshold 无活动多久后清空页面缓存
const idleCacheThreshold = 30 * time.Minute

func cleanupIdlePageCache() {
	cacheRoot := config.GetPagesCacheDir()

	// 有活跃阅读会话（浏览器正在阅读/warmup 中）不动缓存；
	// 但若阅读锁异常未释放（如浏览器强杀、beacon 丢失），
	// 缓存空闲超过 6 小时也强制清理，避免缓存永远残留。
	if isReadingActive() {
		latest, err := latestCacheActivity(cacheRoot)
		if err != nil || time.Since(latest) < 6*time.Hour {
			return
		}
		log.Println("[cache-idle] Reading lock stuck but cache idle >6h, force clearing")
	}
	latest, err := latestCacheActivity(cacheRoot)
	if err != nil {
		return
	}
	if time.Since(latest) < idleCacheThreshold {
		return // 30 分钟内仍有活动（可能刚退出，还保留缓存便于再次秒开）
	}

	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	removed := 0
	for _, e := range entries {
		if os.RemoveAll(filepath.Join(cacheRoot, e.Name())) == nil {
			removed++
		}
	}
	if removed > 0 {
		log.Printf("[cache-idle] Cleared page cache after %.0fm idle: %d entries", idleCacheThreshold.Minutes(), removed)
	}
}

// latestCacheActivity 返回页面缓存目录中最新的文件修改时间。
// 空目录视为"刚活动"，避免每次空转删除日志。
func latestCacheActivity(root string) (time.Time, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return time.Time{}, err
	}
	var latest time.Time
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	if latest.IsZero() {
		return time.Now(), nil
	}
	return latest, nil
}
