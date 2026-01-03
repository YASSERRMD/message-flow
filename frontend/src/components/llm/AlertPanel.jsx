export default function AlertPanel({ alerts }) {
  if (!alerts.length) {
    return null;
  }

  return (
    <section className="alert-panel">
      {alerts.map((alert) => (
        <div key={alert.id} className={`alert alert--${alert.type}`}>
          <span>{alert.message}</span>
          <button type="button" className="ghost">Snooze 24h</button>
        </div>
      ))}
    </section>
  );
}
