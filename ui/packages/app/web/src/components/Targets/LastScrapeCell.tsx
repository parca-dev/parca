import React from 'react';
import {TimeObject, formatDuration, TimeUnits} from '@parca/functions';

const LastScrapeCell = ({
  key,
  lastScrape,
  lastScrapeDuration,
}: {
  key: string;
  lastScrape: TimeObject;
  lastScrapeDuration: TimeObject;
}) => (
  <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-200">
    <p>Last Scrape: {formatDuration(lastScrape, TimeUnits.Nanoseconds)} ago</p>
    <p>Duration: {formatDuration(lastScrapeDuration, TimeUnits.Nanoseconds)}</p>
  </td>
);

export default LastScrapeCell;
