const Capitalize = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1);
};

// function retrieve human readable time frame from datetime
// 1m ago, 20m ago, 1h1m ago, 1h58m ago, 1d1h ago, 1month ago, 1year ago
// function drop seconds and round up to the nearest minute
const TimeAgo = (datetime: string): string => {
  const date = new Date(datetime);
  const now = new Date();
  return Seconds2human(Math.round((now.getTime() - date.getTime()) / 1000));
};

// function retrieve human readable time frame from 2 datetime strings
const TimeDiff = (start: string, end: string): string => {
  const startDate = new Date(start);
  const endDate = new Date(end);
  return Seconds2human(
    Math.round((endDate.getTime() - startDate.getTime()) / 1000)
  );
};

function Seconds2human(seconds: number): string {
  /**
   * Convert a given number of seconds into a human-readable string format.
   *
   * @param seconds - The number of seconds to convert.
   * @returns A string representing the time in a human-readable format.
   *          - If seconds is less than 60, returns the format "{seconds}s".
   *          - If seconds is less than 3600 (1 hour), returns the format "{minutes}m".
   *          - If seconds is 3600 or more, returns the format "{hours}h{minutes}".
   *          - If hours is greater than 0 and minutes is 0, returns the format "{hours}h".
   *          - If the input is null or 0, returns "0".
   */
  if (seconds === null || seconds === 0) {
    return "0";
  } else if (seconds < 60) {
    return `${seconds}s`;
  }

  seconds = Math.floor(seconds / 60);
  const minutes = Math.round(seconds % 60);
  seconds = Math.floor(seconds / 60);
  const hours = Math.round(seconds % 24);

  if (hours > 0) {
    if (minutes === 0) {
      return `${hours}h`;
    } else {
      return `${hours}h${minutes}`;
    }
  } else {
    return `${minutes}m`;
  }
}

function RunStatusColor(status: string): string {
  switch (status) {
    case CRunStatus.completed:
      return "text-success";
    case CRunStatus.started:
      return "text-info";
    case CRunStatus.error:
    case CRunStatus.internal_error:
      return "text-error";
    case CRunStatus.interrupted:
      return "text-warning";
    default:
      return "";
  }
}

export { Capitalize, TimeAgo, TimeDiff, Seconds2human, RunStatusColor };
