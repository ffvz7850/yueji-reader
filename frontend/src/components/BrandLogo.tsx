"use client";

/**
 * 悦集品牌 Logo（SVG 内联，渐变圆角方块 + "悦"字）
 * 在未上传自定义站点图标时作为默认品牌标识，
 * 与 public/icons/icon-*.png 保持同一视觉设计。
 */
export default function BrandLogo({ className = "h-9 w-9" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 48 48"
      className={className}
      aria-hidden="true"
      style={{ display: "block" }}
    >
      <defs>
        <linearGradient id="yueji-brand-g" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#6366f1" />
          <stop offset="100%" stopColor="#8b5cf6" />
        </linearGradient>
      </defs>
      <rect x="1" y="1" width="46" height="46" rx="11" fill="url(#yueji-brand-g)" />
      <text
        x="24"
        y="25"
        textAnchor="middle"
        dominantBaseline="central"
        fontSize="26"
        fontWeight="700"
        fill="#ffffff"
        fontFamily="'PingFang SC','Microsoft YaHei','Noto Sans SC',sans-serif"
      >
        悦
      </text>
    </svg>
  );
}
