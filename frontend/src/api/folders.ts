import { apiClient } from "@/lib/apiClient";
import type { LibraryBrowseResponse } from "@/hooks/useComicTypes";

/**
 * 文件夹式浏览 API
 *
 * GET /api/library-folders?libraryId=xxx&path=sub%2Ffolder
 * 返回指定图库、指定相对目录（path="" 表示库根）下的子文件夹列表（带封面缩略图）。
 * 该目录下的直接作品列表由 /api/comics?folder=... 提供。
 */
export async function fetchLibraryFolders(
  libraryId: string,
  path: string,
  signal?: AbortSignal
): Promise<LibraryBrowseResponse> {
  const params = new URLSearchParams();
  params.set("libraryId", libraryId);
  if (path) params.set("path", path);
  return apiClient.get<LibraryBrowseResponse>(
    `/api/library-folders?${params.toString()}`,
    { signal }
  );
}
