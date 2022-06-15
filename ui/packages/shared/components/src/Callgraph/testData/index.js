export {default as fgSimple} from './fg-simple.json';

export const testData1 = {
  nodes: [
    {
      name: 'start',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 0,
      cum_value: 100,
    },
    {
      name: 'B',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 10,
      cum_value: 80,
    },
    {
      name: 'C',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 5,
      cum_value: 20,
    },
    {
      name: 'D',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 10,
      cum_value: 25,
    },
    {
      name: 'E',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 60,
      cum_value: 60,
    },
  ],
  links: [
    {
      source: 'start',
      target: 'B',
      value: 0,
    },
    {
      source: 'start',
      target: 'C',
      value: 100,
    },
    {
      source: 'B',
      target: 'D',
      value: 50,
    },
    {
      source: 'B',
      target: 'E',
      value: 100,
    },
    {
      source: 'C',
      target: 'D',
      value: 50,
    },
    {
      source: 'E',
      target: 'C',
      value: 100,
    },
    {
      source: 'D',
      target: 'E',
      value: 100,
    },
  ],
};

export const testData2 = {
  nodes: [
    {
      name: 'start',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 0,
      cum_value: 100,
    },
    {
      name: 'B',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 10,
      cum_value: 80,
    },
    {
      name: 'C',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 5,
      cum_value: 20,
    },
    {
      name: 'D',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 10,
      cum_value: 25,
    },
    {
      name: 'E',
      group: 'a',
      line_number: 1234,
      location: 'tralala',
      flat_value: 60,
      cum_value: 60,
    },
  ],
  links: [
    {
      source: 'start',
      target: 'B',
      value: 70,
    },
    {
      source: 'start',
      target: 'C',
      value: 20,
    },
    {
      source: 'B',
      target: 'D',
      value: 10,
    },
    {
      source: 'B',
      target: 'E',
      value: 60,
    },
    {
      source: 'D',
      target: 'C',
      value: 30,
    },
    {
      source: 'C',
      target: 'B',
      value: 15,
    },
  ],
};

export const zherebko = [
  {link: ['1', '2'], reverseDirection: false},
  {link: ['1', '5'], reverseDirection: false},
  {link: ['1', '7'], reverseDirection: false},
  {link: ['2', '3'], reverseDirection: false},
  {link: ['2', '4'], reverseDirection: false},
  {link: ['2', '5'], reverseDirection: false},
  {link: ['2', '7'], reverseDirection: true},
  {link: ['2', '8'], reverseDirection: false},
  {link: ['3', '6'], reverseDirection: false},
  {link: ['3', '8'], reverseDirection: false},
  {link: ['4', '7'], reverseDirection: false},
  {link: ['5', '7'], reverseDirection: false},
  {link: ['5', '8'], reverseDirection: false},
  {link: ['5', '9'], reverseDirection: false},
  {link: ['6', '8'], reverseDirection: false},
  {link: ['7', '8'], reverseDirection: false},
  {link: ['7', '1'], reverseDirection: true},
  {link: ['9', '10'], reverseDirection: false},
  {link: ['9', '11'], reverseDirection: false},
];
