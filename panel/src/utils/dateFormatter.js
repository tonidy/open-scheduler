/**
 * Format a timestamp into a readable date and time string
 * @param {string} timestamp - ISO timestamp string
 * @returns {string} Formatted date string or 'N/A' if invalid
 */
export function formatDate(timestamp) {
  if (!timestamp || timestamp === '0001-01-01T00:00:00Z') return 'N/A';
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: true
  });
}

/**
 * Format a timestamp as relative time (e.g., "2 minutes ago")
 * @param {string} timestamp - ISO timestamp string
 * @returns {string} Relative time string
 */
export function formatEventTimestamp(timestamp) {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now - date;
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  // Show relative time for recent events
  if (diffSecs < 60) {
    return `${diffSecs} second${diffSecs !== 1 ? 's' : ''} ago`;
  } else if (diffMins < 60) {
    return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`;
  } else if (diffHours < 24) {
    return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
  } else if (diffDays < 7) {
    return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
  } else {
    // For older events, show formatted date
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true
    });
  }
}

/**
 * Parse an event string in the format "[timestamp] message"
 * @param {string} event - Event string with timestamp
 * @returns {object} Object with timestamp, message, and formattedTime
 */
export function parseEvent(event) {
  // Parse event format: [2025-11-12T16:31:48+07:00] Message
  const match = event.match(/\[(.*?)\]\s*(.*)/);
  if (match) {
    const timestamp = match[1];
    const message = match[2];
    return {
      timestamp,
      message,
      formattedTime: formatEventTimestamp(timestamp)
    };
  }
  return { timestamp: null, message: event, formattedTime: '' };
}

/**
 * Format a timestamp as a short date (without year if current year)
 * @param {string} timestamp - ISO timestamp string
 * @returns {string} Short formatted date string
 */
export function formatShortDate(timestamp) {
  if (!timestamp || timestamp === '0001-01-01T00:00:00Z') return 'N/A';
  const date = new Date(timestamp);
  const now = new Date();
  const isCurrentYear = date.getFullYear() === now.getFullYear();

  return date.toLocaleString('en-US', {
    year: isCurrentYear ? undefined : 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: true
  });
}

