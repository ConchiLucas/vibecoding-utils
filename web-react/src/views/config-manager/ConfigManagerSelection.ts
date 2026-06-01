type ConnectionLike = {
  ID: number;
};

export function resolveSelectedConnectionId(
  selectedConnectionId: number | null,
  connections: ConnectionLike[],
) {
  if (connections.length === 0) return null;
  if (selectedConnectionId !== null && connections.some(conn => conn.ID === selectedConnectionId)) {
    return selectedConnectionId;
  }
  return connections[0].ID;
}
