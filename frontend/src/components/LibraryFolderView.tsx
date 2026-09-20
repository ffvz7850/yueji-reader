"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  Folder,
  FolderOpen,
  ArrowUp,
  ChevronRight,
  Home,
  Library as LibraryIcon,
} from "lucide-react";
import { apiPath } from "@/lib/base-path";
import { ssCacheGet, ssCacheSet } from "@/lib/ss-cache";
import type { Library } from "@/api/libraries";
import type { LibraryBrowseResponse } from "@/hooks/useComicTypes";
import { naturalSortKey } from "@/lib/comic-utils";
import { useTranslation } from "@/lib/i18n";
import NSFWCoverGuard from "@/components/NSFWCoverGuard";
import { hasNSFWTitle } from "@/lib/nsfw";

/** 合并网格中的漫画条目（取父页面 useComics 返回的字段） */
export interface FolderViewComic {
  id: string;
  title: string;
  coverUrl?: string | null;
  pageCount?: number;
  progress?: number;
  isFavorite?: boolean;
  type?: string;
}

/**
 * 文件夹浏览数据的会话级缓存（key = libraryId|folderPath）。
 * 从详情页/阅读器返回目录时直接渲染缓存内容，后台静默刷新，避免"闪一下"。
 * 同时写入 sessionStorage：手机返回整页重载（bfcache 被弹出）时数据仍在，
 * 首帧立即渲染不闪。
 */
const folderViewCache = new Map<string, LibraryBrowseResponse>();
const SS_BUCKET = "folders";
const SS_MAX_ENTRIES = 12;

/**
 * LibraryFolderView 文件夹式浏览视图
 *
 * 把图库按目录组织成树：库根 → 子文件夹 → 作品。
 * - activeLibraryId 为空时显示"图库选择"卡片；
 * - 进入图库后显示面包屑 + 合并网格（子文件夹卡片 + 当前目录直接作品卡片混排，
 *   按名称自然排序，不再分"文件夹区/图集区"两个区块）。
 */
export default function LibraryFolderView({
  libraries,
  activeLibraryId,
  folderPath,
  onNavigate,
  onReset,
  privacyEnabled,
  blurNSFW,
  refreshKey = 0,
  comics = [],
  comicsLoading = false,
}: {
  libraries: Library[];
  activeLibraryId: string | null;
  folderPath: string;
  onNavigate: (libraryId: string, folderPath: string) => void;
  onReset: () => void;
  privacyEnabled: boolean;
  blurNSFW: boolean;
  refreshKey?: number;
  comics?: FolderViewComic[];
  comicsLoading?: boolean;
}) {
  const t = useTranslation();
  const cacheKey = activeLibraryId ? `${activeLibraryId}|${folderPath}` : "";
  // 有缓存时直接渲染缓存内容（返回场景不闪），后台再静默刷新
  const [data, setData] = useState<LibraryBrowseResponse | null>(() => {
    if (!cacheKey) return null;
    return folderViewCache.get(cacheKey) ?? ssCacheGet<LibraryBrowseResponse>(SS_BUCKET, cacheKey) ?? null;
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const activeLibrary = libraries.find((l) => l.id === activeLibraryId) || null;

  useEffect(() => {
    // 未选择图库时不请求
    if (!activeLibraryId || !activeLibrary) return;
    if (abortRef.current) abortRef.current.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    // 有缓存时静默刷新（返回场景不闪）；首次加载或显式刷新（refreshKey>0）才显示 loading
    if (!folderViewCache.has(cacheKey) || refreshKey > 0) {
      setLoading(true);
    }
    setError(false);

    const params = new URLSearchParams();
    params.set("libraryId", activeLibraryId);
    if (folderPath) params.set("path", folderPath);

    fetch(apiPath(`/api/library-folders?${params.toString()}`), {
      signal: controller.signal,
      cache: "no-store",
    })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json() as Promise<LibraryBrowseResponse>;
      })
      .then((d) => {
        setData(d);
        folderViewCache.set(cacheKey, d);
        ssCacheSet(SS_BUCKET, cacheKey, d, SS_MAX_ENTRIES);
        setLoading(false);
      })
      .catch((err) => {
        if (err?.name === "AbortError") return;
        setError(true);
        setLoading(false);
      });
    return () => {
      controller.abort();
    };
  }, [activeLibraryId, activeLibrary, folderPath, refreshKey, cacheKey]);

  // 没有选中图库 → 图库选择页
  if (!activeLibraryId || !activeLibrary) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-1.5 text-sm text-muted">
          <LibraryIcon className="h-4 w-4" />
          <span>{t.folder?.chooseLibrary || "选择一个图库开始浏览"}</span>
        </div>
        {libraries.length === 0 ? (
          <div className="dashboard-glass rounded-xl p-10 text-center">
            <p className="text-sm text-muted">{t.folder?.noLibraries || "没有可访问的图库"}</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-2.5 sm:gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8">
            {libraries.map((lib) => (
              <button
                key={lib.id}
                onClick={() => onNavigate(lib.id, "")}
                className="group flex flex-col items-center justify-center gap-3 rounded-xl border border-border/40 bg-card/60 p-6 text-center transition-all hover:border-accent/40 hover:bg-card"
              >
                <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-accent/10 text-accent transition-transform group-hover:scale-105">
                  <FolderOpen className="h-8 w-8" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-foreground">{lib.name}</p>
                  <p className="mt-0.5 text-[11px] text-muted">
                    {lib.comicCount ?? 0} {t.folder?.albums || "本"}
                  </p>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    );
  }

  const segments = data?.segments || [];
  const folders = data?.folders || [];

  // 合并网格条目：文件夹卡片 + 当前目录直接作品卡片，按名称自然排序混排
  type GridItem =
    | { kind: "folder"; key: string; name: string; coverUrl?: string | null; itemCount: number; albumCount: number; path: string }
    | { kind: "comic"; key: string; name: string; coverUrl?: string | null; pageCount: number; progress: number; id: string; type?: string };
  const gridItems: GridItem[] = [
    ...folders.map((f) => ({
      kind: "folder" as const,
      key: `f:${f.path}`,
      name: f.name,
      coverUrl: f.coverUrl,
      itemCount: f.itemCount,
      albumCount: f.albumCount,
      path: f.path,
    })),
    ...comics.map((c) => ({
      kind: "comic" as const,
      key: `c:${c.id}`,
      name: c.title,
      coverUrl: c.coverUrl,
      pageCount: c.pageCount || 0,
      progress: c.progress || 0,
      id: c.id,
      type: c.type,
    })),
  ].sort((a, b) => naturalSortKey(a.name).localeCompare(naturalSortKey(b.name)));

  const isEmpty = folders.length === 0 && comics.length === 0;

  return (
    <div className="space-y-3">
      {/* 面包屑 + 上级 */}
      <div className="flex min-w-0 items-center gap-1.5 overflow-x-auto scrollbar-hide text-sm">
        <button
          onClick={onReset}
          className="flex shrink-0 items-center gap-1 rounded-lg px-2 py-1 text-muted transition-colors hover:bg-card hover:text-foreground"
          title={t.folder?.backToLibraries || "返回图库列表"}
        >
          <Home className="h-3.5 w-3.5" />
          <span className="hidden sm:inline truncate max-w-[120px]">{data?.libraryName || activeLibrary?.name}</span>
        </button>
        {segments.map((seg, i) => {
          const target = segments.slice(0, i + 1).join("/");
          return (
            <span key={seg} className="flex min-w-0 items-center gap-1.5">
              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted/50" />
              <button
                onClick={() => onNavigate(activeLibraryId, target)}
                className="max-w-[160px] truncate rounded-lg px-2 py-1 text-foreground/80 transition-colors hover:bg-card hover:text-foreground"
              >
                {seg}
              </button>
            </span>
          );
        })}
        {segments.length === 0 && (
          <>
            <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted/50" />
            <span className="shrink-0 px-1 text-muted">{t.folder?.root || "根目录"}</span>
          </>
        )}
        {(data?.parentPath || segments.length > 0) && (
          <button
            onClick={() =>
              data?.parentPath
                ? onNavigate(activeLibraryId, data.parentPath)
                : onNavigate(activeLibraryId, "")
            }
            className="ml-auto flex shrink-0 items-center gap-1 rounded-lg border border-border/40 bg-card px-2 py-1 text-xs text-muted transition-colors hover:text-foreground"
          >
            <ArrowUp className="h-3.5 w-3.5" />
            {t.folder?.up || "上级"}
          </button>
        )}
      </div>

      {/* 合并网格：子文件夹 + 直接作品混排 */}
      {!data && loading ? (
        <div className="grid grid-cols-2 gap-2.5 sm:gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8">
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} className="overflow-hidden rounded-xl bg-card">
              <div className="skeleton-shimmer aspect-[5/7] w-full" />
              <div className="space-y-2 p-3">
                <div className="skeleton-shimmer h-4 w-3/4 rounded" />
                <div className="skeleton-shimmer h-3 w-1/2 rounded" />
              </div>
            </div>
          ))}
        </div>
      ) : error ? (
        <div className="dashboard-glass rounded-xl p-10 text-center">
          <p className="text-sm text-muted">{t.common?.forbiddenDesc || "加载失败，请重试"}</p>
        </div>
      ) : isEmpty ? (
        <div className="dashboard-glass rounded-xl p-10 text-center">
          <p className="text-sm text-muted">{t.folder?.empty || "该目录为空"}</p>
          <p className="mt-1 text-xs text-muted/60">{t.folder?.emptyHint || "把作品放到这个目录下并扫描后即可在此看到"}</p>
        </div>
      ) : (
        <div className={`grid grid-cols-2 gap-2.5 sm:gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8 transition-opacity duration-200 ${loading || comicsLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
          {gridItems.map((item) =>
            item.kind === "folder" ? (
              <button
                key={item.key}
                onClick={() => onNavigate(activeLibraryId, item.path)}
                className="group block text-left"
              >
                <div className="relative aspect-[5/7] w-full overflow-hidden rounded-xl bg-card/50 backdrop-blur-sm border border-border/30 cover-glow transition-transform duration-300 group-hover:scale-[1.02]">
                  {item.coverUrl ? (
                    <NSFWCoverGuard
                      src={item.coverUrl}
                      alt={item.name}
                      isNSFW={hasNSFWTitle(item.name)}
                      blurEnabled={privacyEnabled && blurNSFW}
                      fill
                      unoptimized
                      className="object-cover"
                      sizes="(max-width: 640px) 50vw, 16vw"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center bg-card">
                      <Folder className="h-10 w-10 text-muted/40" />
                    </div>
                  )}
                  <div className="absolute left-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-lg bg-black/55 backdrop-blur-sm border border-white/10">
                    <Folder className="h-3.5 w-3.5 text-amber-300/90" />
                  </div>
                  <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent p-2 pt-8">
                    <p className="text-[11px] font-medium text-white/90 line-clamp-1 drop-shadow">
                      {item.name}
                    </p>
                  </div>
                </div>
                <p className="mt-1.5 px-0.5 text-[11px] text-muted">
                  {item.itemCount} {t.folder?.items || "项"}
                  {item.albumCount > 0 ? ` · ${item.albumCount} ${t.folder?.albums || "本"}` : ""}
                </p>
              </button>
            ) : (
              <Link
                key={item.key}
                href={`/reader/${item.id}`}
                className="group block text-left"
              >
                <div className="relative aspect-[5/7] w-full overflow-hidden rounded-xl bg-card/50 backdrop-blur-sm border border-border/30 cover-glow transition-transform duration-300 group-hover:scale-[1.02]">
                  {item.coverUrl ? (
                    <NSFWCoverGuard
                      src={item.coverUrl}
                      alt={item.name}
                      isNSFW={hasNSFWTitle(item.name)}
                      blurEnabled={privacyEnabled && blurNSFW}
                      fill
                      unoptimized
                      className="object-cover"
                      sizes="(max-width: 640px) 50vw, 16vw"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center bg-card">
                      <Folder className="h-10 w-10 text-muted/40" />
                    </div>
                  )}
                  {item.progress > 0 && (
                    <div className="absolute inset-x-0 top-0 h-0.5 bg-black/40">
                      <div className="h-full bg-accent" style={{ width: `${Math.min(100, Math.round(item.progress))}%` }} />
                    </div>
                  )}
                  <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent p-2 pt-8">
                    <p className="text-[11px] font-medium text-white/90 line-clamp-1 drop-shadow">
                      {item.name}
                    </p>
                  </div>
                </div>
                <p className="mt-1.5 px-0.5 text-[11px] text-muted">
                  {item.pageCount > 0 ? `${item.pageCount} 页` : t.folder?.items || "项"}
                </p>
              </Link>
            )
          )}
        </div>
      )}
    </div>
  );
}
