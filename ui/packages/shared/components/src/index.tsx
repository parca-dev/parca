// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React from 'react';
import Button from './Button';
import ButtonGroup from './ButtonGroup';
import Card from './Card';
import DateTimeRangePicker, {DateTimeRange} from './DateTimeRangePicker';
import DateTimePicker from './DateTimePicker';
import Dropdown from './Dropdown';
import GrpcMetadataContext, {GrpcMetadataProvider, useGrpcMetadata} from './GrpcMetadataContext';
import Input from './Input';
import GraphTooltip from './GraphTooltip';
import MatchersInput from './MatchersInput';
import MetricsGraph from './MetricsGraph';
import Modal from './Modal';
import ParcaThemeContext, {ParcaThemeProvider, useParcaTheme} from './ParcaThemeContext';
import Pill, {PillVariant} from './Pill';
import ProfileExplorer from './ProfileExplorer';
import ProfileMetricsGraph from './ProfileMetricsGraph';
import ProfileSelector from './ProfileSelector';
import ResponsiveSvg from './ResponsiveSvg';
import Select from './Select';
import type {SelectElement} from './Select';
import SearchNodes from './SearchNodes';
import Spinner from './Spinner';
import Tab from './Tab';
import EmptyState from './EmptyState';

const Callgraph = React.lazy(async () => await import('./Callgraph'));

export type {PillVariant, SelectElement};

export {
  Button,
  ButtonGroup,
  Callgraph,
  Card,
  DateTimePicker,
  DateTimeRange,
  DateTimeRangePicker,
  Dropdown,
  GraphTooltip,
  GrpcMetadataContext,
  GrpcMetadataProvider,
  Input,
  MatchersInput,
  MetricsGraph,
  Modal,
  ParcaThemeContext,
  ParcaThemeProvider,
  Pill,
  ProfileExplorer,
  ProfileMetricsGraph,
  ProfileSelector,
  ResponsiveSvg,
  Select,
  SearchNodes,
  Spinner,
  Tab,
  EmptyState,
  useGrpcMetadata,
  useParcaTheme,
};
