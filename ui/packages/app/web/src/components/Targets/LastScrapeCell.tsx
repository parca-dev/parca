import React from 'react';
import {TimeObject, formatDuration, TimeUnits, convertTime} from '@parca/functions';

const LastScrapeCell = ({
  key,
  lastScrape,
  lastScrapeDuration,
}: {
  key: string;
  lastScrape: TimeObject;
  lastScrapeDuration: TimeObject;
}) => {
  const nowInNanoseconds = convertTime(
    new Date().getTime(),
    TimeUnits.Milliseconds,
    TimeUnits.Nanos
  );
  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-200">
      <p>Last Scrape: {formatDuration(lastScrape, nowInNanoseconds)} ago</p>
      <p>Duration: {formatDuration(lastScrapeDuration)}</p>
    </td>
  );
};

export default LastScrapeCell;
