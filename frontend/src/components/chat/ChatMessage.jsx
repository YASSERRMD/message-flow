import { extractMedia, shouldHidePlaceholderText } from "./messageUtils.js";
import { getAvatarStyle } from "./avatar.js";

export default function ChatMessage({ message, isGroup, formatTime }) {
  const media = extractMedia(message.metadata_json);
  const hideText = shouldHidePlaceholderText(message.content);
  const senderName = message.sender_name || (message.sender || "").split("@")[0] || "Unknown";
  const isSenderUnknown = senderName === "Unknown" || senderName === "" || senderName === "?";

  return (
    <div className={`message ${message.is_outbound ? "outbound" : ""}`}>
      <div className="message-bubble">
        {!message.is_outbound && isGroup && !isSenderUnknown && (
          <div
            className="message-sender"
            style={{
              color: getAvatarStyle(senderName).color,
              fontSize: "12px",
              fontWeight: 600,
              marginBottom: "4px"
            }}
          >
            {senderName}
          </div>
        )}

        {media?.has_media && (
          <div className={`wa-media wa-media-${media.media_type}`}>
            {media.media_type === "image" && (
              <div className="media-placeholder">
                <span className="media-icon">📷</span>
                <span className="media-label">Image</span>
              </div>
            )}
            {media.media_type === "video" && (
              <div className="media-placeholder">
                <span className="media-icon">🎬</span>
                <span className="media-label">
                  Video
                  {media.duration_seconds
                    ? ` (${Math.floor(media.duration_seconds / 60)}:${String(media.duration_seconds % 60).padStart(2, "0")})`
                    : ""}
                </span>
              </div>
            )}
            {media.media_type === "audio" && (
              <div className="media-placeholder">
                <span className="media-icon">🎵</span>
                <span className="media-label">
                  Audio
                  {media.duration_seconds
                    ? ` (${Math.floor(media.duration_seconds / 60)}:${String(media.duration_seconds % 60).padStart(2, "0")})`
                    : ""}
                </span>
              </div>
            )}
            {media.media_type === "document" && (
              <div className="media-placeholder">
                <span className="media-icon">📄</span>
                <span className="media-label">{media.file_name || "Document"}</span>
              </div>
            )}
            {media.media_type === "sticker" && (
              <div className="media-placeholder">
                <span className="media-icon">🎭</span>
                <span className="media-label">Sticker</span>
              </div>
            )}
          </div>
        )}

        {!hideText && (
          <div className="message-text">{message.content}</div>
        )}

        <div className="message-meta">
          <span className="message-time">{formatTime(message.timestamp || message.created_at)}</span>
          {message.is_outbound && (
            <span className="message-status">
              <i className="fas fa-check-double" style={{ color: "#53bdeb" }}></i>
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
