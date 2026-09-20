"use client";

/**
 * sessionStorage 持久化 LRU 缓存
 *
 * 背景：手机手势返回依赖 iOS bfcache（后退缓存），但 bfcache 只能保留最近
 * 少数页面，深层文件夹嵌套、图片多内存压力大时上层页面会被弹出，返回变成
 * 整页重新加载，模块级内存缓存（Map）全部丢失 → 文件夹列表重新加载。
 *
 * 方案：把数据缓存同时写入 sessionStorage（刷新/整页加载后仍在），返回时
 * 首帧即读到缓存数据直接渲染，无骨架屏闪烁；网络请求只在后台静默刷新。
 * sessionStorage 在同一个标签页会话内持久，容量约 5MB，用 LRU 限制条数。
 */
const PREFIX = "nwr:sscache:v1:";

/** 读取持久化缓存；不存在或解析失败返回 null */
export function ssCacheGet<T>(bucket: string, key: string): T | null {
  try {
    const raw = sessionStorage.getItem(PREFIX + bucket + ":" + key);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { v: T };
    return parsed.v;
  } catch {
    return null;
  }
}

/** 写入持久化缓存（LRU：先移除旧键再写，保持插入序；超限淘汰最旧） */
export function ssCacheSet<T>(bucket: string, key: string, value: T, maxEntries = 8) {
  try {
    const fullKey = PREFIX + bucket + ":" + key;
    sessionStorage.removeItem(fullKey);
    sessionStorage.setItem(fullKey, JSON.stringify({ v: value }));
    // sessionStorage 迭代顺序即插入顺序：超限时淘汰最旧
    const keys: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const k = sessionStorage.key(i);
      if (k && k.startsWith(PREFIX + bucket + ":")) keys.push(k);
    }
    while (keys.length > maxEntries) {
      const oldest = keys.shift();
      if (oldest) sessionStorage.removeItem(oldest);
    }
  } catch {
    // 容量超限/隐私模式：静默忽略，内存缓存仍生效
  }
}

/** 清除某个 bucket 的全部持久化缓存 */
export function ssCacheClearBucket(bucket: string) {
  try {
    const keys: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const k = sessionStorage.key(i);
      if (k && k.startsWith(PREFIX + bucket + ":")) keys.push(k);
    }
    keys.forEach((k) => sessionStorage.removeItem(k));
  } catch {
    // ignore
  }
}
