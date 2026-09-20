# 悦集 YueJi

> 自托管图集 / 漫画浏览器，Docker 一键部署。
> 基于 [cropflre/nowen-reader](https://github.com/cropflre/nowen-reader) 深度改造，专为「文件夹散图」图库优化。

悦集把 Nowen Reader 改造成一个**文件夹样式浏览**的图集阅读器，非常适合管理大量子目录写真集 / 图集：文件夹封面缩略图、手机无感返回、长条模式无感刷图，开箱即用。

## ✨ 特性

- **文件夹样式浏览**：子目录图库按文件夹展示，文件夹封面显示缩略图，文件夹与图集合并显示，不再全铺一屏
- **无感刷图流**：长条模式按作者原版加载策略 + 参数调优（预加载 12 张 / 视口前后 7 页渲染 / 20 页 DOM 缓冲），滚动零等待
- **手机无感返回**：从阅读页返回文件夹列表不重新加载、不跳回顶端，多层嵌套同样支持
- **双击放大禁用**：滑动浏览不再误触缩放
- **阅读页磁盘缓存默认关闭**：文件夹散图直读源文件，不产生无用缓存（压缩包/PDF 图集可开 `PAGE_CACHE_ENABLED=true` 恢复秒开）
- **图片字节缓存**：响应带 `Cache-Control: immutable`，回翻秒显
- 其余原版能力完整保留：双页 / 单页 / 长条三种阅读模式、PDF 支持、WebDAV 同步、AI 功能等

## 🐳 Docker 部署

```yaml
services:
  yueji-reader:
    image: ffvz/nowen-reader:latest
    container_name: yueji-reader
    restart: unless-stopped
    ports:
      - "6680:3000"
    volumes:
      - ./data:/data                 # 数据持久化（数据库）
      - ./cache:/app/.cache          # 缓存目录
      - /vol2/1000/图库/photo:/app/comics   # 漫画/图集目录，改成你的实际路径
    environment:
      - PUID=0
      - PGID=0
      - GIN_MODE=release
      - DATABASE_URL=/data/nowen-reader.db
      - COMICS_DIR=/app/comics
      - DATA_DIR=/app/.cache
      - PORT=3000
      - TZ=Asia/Shanghai
      # - PAGE_CACHE_ENABLED=true    # 需要压缩包/PDF 图集秒开时开启阅读页缓存（默认关闭）
```

> 首次启动后访问 `http://<NAS-IP>:6680`，创建管理员账号即可。图库目录下所有子文件夹 / 压缩包会自动扫描入库。

## 🔧 阅读设置

- 阅读页 → 设置 → **预加载图片数量**（0~20）：长条浏览时提前下载的图片数，默认 12，嫌卡可调大，内存紧张可调小
- 长条 / 单页 / 双页三种阅读模式随时切换

## 🏗 本地构建

```bash
# 前端
cd frontend && npm install && npm run build
# 嵌入 web/dist（go:embed）
node frontend/scripts/copy-dist.mjs
# 后端（交叉编译 linux/amd64）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist-linux/nowen-reader-linux-amd64 ./cmd/server
```

## 📦 版本

当前版本：**0.4.9**（加载参数优化版）

版本演进摘要：

| 版本 | 说明 |
|---|---|
| 0.4.3 | 长条模式参数化预加载（预加载 6 张 + 12 屏窗口） |
| 0.4.5 | 阅读页缓存闲置自动清理（30 分钟无活动清空） |
| 0.4.6 | 解码队列全量入队渐进预解码（后经实测否决） |
| 0.4.7 | 阅读页磁盘缓存默认关闭（`PAGE_CACHE_ENABLED` 开关） |
| 0.4.8 | 加载策略整体回退作者原版（虚拟化 + 原生预加载） |
| 0.4.9 | 仅调参数：预加载 12 / 渲染 7 / DOM 缓冲 20 / eager 5 |

## 📄 协议

本项目基于 [cropflre/nowen-reader](https://github.com/cropflre/nowen-reader) 改造，遵循 **GPL-3.0** 协议，版权归原作者与改造者所有。

## 🙏 致谢

- [cropflre/nowen-reader](https://github.com/cropflre/nowen-reader) —— 优秀的自托管漫画阅读器
