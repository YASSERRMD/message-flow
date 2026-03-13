import { useEffect } from "react";

function MediaStage({ item }) {
  if (!item) return null;

  switch (item.type) {
    case "image":
    case "sticker":
      return (
        <img
          src={item.url}
          alt={item.caption || item.fileName}
          className={`media-viewer-asset ${item.type === "sticker" ? "sticker" : ""}`}
        />
      );
    case "video":
      return (
        <video className="media-viewer-video" controls autoPlay preload="metadata">
          <source src={item.url} type={item.media.mime_type || "video/mp4"} />
          Your browser does not support video playback.
        </video>
      );
    case "audio":
      return (
        <div className="media-viewer-audio-shell">
          <div className="media-viewer-audio-icon">
            <i className="fas fa-wave-square"></i>
          </div>
          <audio className="media-viewer-audio" controls preload="metadata">
            <source src={item.url} type={item.media.mime_type || "audio/ogg"} />
            Your browser does not support audio playback.
          </audio>
        </div>
      );
    case "document":
      return (
        <div className="media-viewer-document-shell">
          <div className="media-viewer-document-icon">
            <i className="fas fa-file-lines"></i>
          </div>
          <h4>{item.fileName}</h4>
          <p>Open or download this document from the action panel.</p>
        </div>
      );
    default:
      return (
        <div className="media-viewer-document-shell">
          <div className="media-viewer-document-icon">
            <i className={`fas ${item.icon}`}></i>
          </div>
          <h4>{item.fileName}</h4>
        </div>
      );
  }
}

function formatDateTime(timestamp) {
  if (!timestamp) return "Unknown time";
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "Unknown time";
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  });
}

export default function MediaViewerModal({
  item,
  items = [],
  activeIndex = 0,
  onClose,
  onPrev,
  onNext,
  onSelect,
  onLocate
}) {
  useEffect(() => {
    if (!item) return undefined;

    const handleKeyDown = (event) => {
      if (event.key === "Escape") onClose?.();
      if (event.key === "ArrowLeft") onPrev?.();
      if (event.key === "ArrowRight") onNext?.();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [item, onClose, onPrev, onNext]);

  if (!item) return null;

  const metaChips = [item.label, item.durationLabel, item.sizeLabel].filter(Boolean);

  return (
    <div className="media-viewer-overlay" onClick={onClose}>
      <div className="media-viewer-modal" onClick={(event) => event.stopPropagation()}>
        <div className="media-viewer-topbar">
          <div>
            <span className="media-viewer-kicker">Media viewer</span>
            <h3>{item.fileName}</h3>
          </div>
          <button type="button" className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="media-viewer-layout">
          <div className="media-viewer-stage-shell">
            {items.length > 1 && (
              <button
                type="button"
                className="media-viewer-nav prev"
                onClick={onPrev}
                disabled={activeIndex <= 0}
              >
                <i className="fas fa-chevron-left"></i>
              </button>
            )}

            <div className="media-viewer-stage">
              {item.url ? (
                <MediaStage item={item} />
              ) : (
                <div className="media-viewer-document-shell">
                  <div className="media-viewer-document-icon">
                    <i className={`fas ${item.icon}`}></i>
                  </div>
                  <h4>{item.fileName}</h4>
                  <p>This media file is not available on disk yet.</p>
                </div>
              )}
            </div>

            {items.length > 1 && (
              <button
                type="button"
                className="media-viewer-nav next"
                onClick={onNext}
                disabled={activeIndex >= items.length - 1}
              >
                <i className="fas fa-chevron-right"></i>
              </button>
            )}
          </div>

          <aside className="media-viewer-sidebar">
            <div className="media-viewer-sidebar-card">
              <div className="media-viewer-type-row">
                <span className="media-chip"><i className={`fas ${item.icon}`}></i>{item.label}</span>
                <span className="media-chip subtle">{activeIndex + 1} / {items.length}</span>
              </div>

              {item.caption && <p className="media-viewer-caption">{item.caption}</p>}

              <div className="media-viewer-meta-grid">
                <div className="media-meta-item">
                  <span>From</span>
                  <strong>{item.senderName}</strong>
                </div>
                <div className="media-meta-item">
                  <span>Sent</span>
                  <strong>{formatDateTime(item.timestamp)}</strong>
                </div>
                <div className="media-meta-item">
                  <span>Format</span>
                  <strong>{item.media.mime_type || item.label}</strong>
                </div>
                <div className="media-meta-item">
                  <span>Details</span>
                  <strong>{metaChips.join(" • ") || "No extra details"}</strong>
                </div>
              </div>
            </div>

            <div className="media-viewer-sidebar-card">
              <h4>Actions</h4>
              <div className="media-viewer-actions">
                {item.url && (
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="media-action-btn"
                  >
                    <i className="fas fa-up-right-from-square"></i>
                    Open original
                  </a>
                )}
                {item.url && (
                  <a
                    href={item.url}
                    download={item.fileName}
                    className="media-action-btn"
                  >
                    <i className="fas fa-download"></i>
                    Download
                  </a>
                )}
                <button
                  type="button"
                  className="media-action-btn"
                  onClick={() => onLocate?.(item.id)}
                >
                  <i className="fas fa-location-crosshairs"></i>
                  Locate in chat
                </button>
              </div>
            </div>
          </aside>
        </div>

        {items.length > 1 && (
          <div className="media-viewer-strip">
            {items.map((candidate, index) => (
              <button
                key={candidate.id}
                type="button"
                className={`media-viewer-thumb ${candidate.id === item.id ? "active" : ""}`}
                onClick={() => onSelect?.(candidate.id)}
              >
                {candidate.url && (candidate.type === "image" || candidate.type === "sticker") ? (
                  <img src={candidate.url} alt={candidate.fileName} />
                ) : (
                  <span><i className={`fas ${candidate.icon}`}></i></span>
                )}
                <small>{index + 1}</small>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
