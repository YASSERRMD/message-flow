import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function CommentThread({ actionItemId, token, csrf }) {
  const [comments, setComments] = useState([]);
  const [content, setContent] = useState("");

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadComments = async () => {
    const response = await fetch(`${API_BASE}/action-items/${actionItemId}/comments`, { headers });
    if (response.ok) {
      const data = await response.json();
      setComments(data.data || []);
    }
  };

  useEffect(() => {
    if (!token || !actionItemId) return;
    loadComments();
  }, [token, actionItemId]);

  const submit = async (event) => {
    event.preventDefault();
    if (!content) return;
    await fetch(`${API_BASE}/action-items/${actionItemId}/comments`, {
      method: "POST",
      headers,
      body: JSON.stringify({ content })
    });
    setContent("");
    loadComments();
  };

  return (
    <div className="comment-thread">
      <div className="comment-list">
        {comments.map((comment) => (
          <div key={comment.id} className="comment">
            <strong>User {comment.user_id}</strong>
            <p>{comment.content}</p>
            <span>{new Date(comment.created_at).toLocaleString()}</span>
          </div>
        ))}
      </div>
      <form className="comment-form" onSubmit={submit}>
        <input
          placeholder="Write a comment"
          value={content}
          onChange={(event) => setContent(event.target.value)}
        />
        <button className="ghost" type="submit">Post</button>
      </form>
    </div>
  );
}
