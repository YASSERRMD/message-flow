const parseKeyPoints = (value) => {
  if (!value) return null;
  if (typeof value === "object") return value;
  try {
    return JSON.parse(value);
  } catch (error) {
    return null;
  }
};

// Component for displaying AI-generated daily summaries
export default function DailySummaryCard({ summary, stats }) {
  const keyPoints = parseKeyPoints(summary?.key_points_json);
  const topics = keyPoints?.topics || [];
  const sentiment = keyPoints?.sentiment || "Neutral";

  return (
    <section className="panel daily-summary">
      <header>
        <h3>Daily Summary</h3>
        <span>{summary ? new Date(summary.created_at).toLocaleDateString() : "--"}</span>
      </header>
      <div className="panel-body">
        {summary ? (
          <>
            <p className="summary-text">{summary.summary_text}</p>
            <div className="summary-metrics">
              <div>
                <p>Sentiment</p>
                <strong>{sentiment}</strong>
              </div>
              <div>
                <p>Conversations</p>
                <strong>{stats.total_conversations}</strong>
              </div>
              <div>
                <p>Important</p>
                <strong>{stats.important_messages}</strong>
              </div>
            </div>
            <div className="summary-topics">
              <p>Topics</p>
              <div className="chip-row">
                {topics.length ? topics.map((topic) => (
                  <span key={topic} className="chip">{topic}</span>
                )) : <span className="empty">No topics yet.</span>}
              </div>
            </div>
          </>
        ) : (
          <p className="empty">No summary generated yet.</p>
        )}
      </div>
    </section>
  );
}
