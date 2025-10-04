const Capitalize = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1);
};

// function retrieve human readable time frame from datetime
// 1m ago, 20m ago, 1h1m ago, 1h58m ago, 1d1h ago, 1month ago, 1year ago
// function drop seconds and round up to the nearest minute
// handles clock skew by clamping negative values (future dates) to 0
const TimeAgo = (datetime: string): string => {
  const date = new Date(datetime);
  const now = new Date();
  const diff = Math.round((now.getTime() - date.getTime()) / 1000);

  // Clamp negative values to 0 (clock skew tolerance for future dates)
  return Seconds2human(diff < 0 ? 0 : diff);
};

// function retrieve human readable time frame from 2 datetime strings
// handles clock skew by clamping negative differences to 0
const TimeDiff = (start: string, end: string): string => {
  const startDate = new Date(start);
  const endDate = new Date(end);
  const diff = Math.round((endDate.getTime() - startDate.getTime()) / 1000);

  // Clamp negative values to 0 (end before start due to clock skew)
  return Seconds2human(diff < 0 ? 0 : diff);
};

const Seconds2human = (seconds: number): string => {
  /**
   * Convert a given number of seconds into a human-readable string format.
   *
   * @param seconds - The number of seconds to convert.
   * @returns A string representing the time in a human-readable format with progressive granularity:
   *          - < 60s: seconds only (1s, 30s, 59s)
   *          - >= 60s && < 5m: minutes+seconds (1m0s, 3m30s, 4m59s)
   *          - >= 5m && < 60m: minutes only (5m, 30m, 59m)
   *          - >= 60m && < 4h: hours+minutes (1h, 2h30m, 3h59m)
   *          - >= 4h && < 1d: hours only (4h, 12h, 23h)
   *          - >= 1d && < 4d: days+hours (1d, 2d12h, 3d23h)
   *          - >= 4d && < 60d: days only (4d, 30d, 59d)
   *          - >= 60d && < 365d: months only (2mo, 6mo, 12mo)
   *          - >= 365d: years+months (1y, 2y6mo, 10y3mo)
   */
  if (seconds === null || seconds === 0) {
    return "0";
  }

  // < 60s: seconds only
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;

  // >= 60s && < 5m: minutes+seconds
  if (minutes < 5) {
    return remainingSeconds === 0 ? `${minutes}m` : `${minutes}m${remainingSeconds}s`;
  }

  // >= 5m && < 60m: minutes only
  if (minutes < 60) {
    return `${minutes}m`;
  }

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;

  // >= 60m && < 4h: hours+minutes
  if (hours < 4) {
    return remainingMinutes === 0 ? `${hours}h` : `${hours}h${remainingMinutes}m`;
  }

  // >= 4h && < 1d: hours only
  if (hours < 24) {
    return `${hours}h`;
  }

  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;

  // >= 1d && < 4d: days+hours
  if (days < 4) {
    return remainingHours === 0 ? `${days}d` : `${days}d${remainingHours}h`;
  }

  // >= 4d && < 60d: days only
  if (days < 60) {
    return `${days}d`;
  }

  // >= 60d && < 365d: months only (approximate: 30 days/month)
  if (days < 365) {
    const months = Math.floor(days / 30);
    return `${months}mo`;
  }

  // >= 365d: years+months
  const years = Math.floor(days / 365);
  const remainingDays = days % 365;
  const remainingMonths = Math.floor(remainingDays / 30);
  return remainingMonths === 0 ? `${years}y` : `${years}y${remainingMonths}mo`;
};

const RunStatusClass = (status: string): string => {
  switch (status) {
    case CRunStatus.completed:
      return "badge badge-success bg-opacity-60";
    case CRunStatus.started:
      return "badge badge-info bg-opacity-60";
    case CRunStatus.error:
    case CRunStatus.internal_error:
      return "badge badge-error bg-opacity-60";
    case CRunStatus.interrupted:
      return "badge badge-warning bg-opacity-60";
    default:
      return "";
  }
};

const UpdateTitle = (title?: string): void => {
  document.title = title ? title : "ScriptFlow";
};

export { Capitalize, RunStatusClass, Seconds2human, TimeAgo, TimeDiff, UpdateTitle };
