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

import {Button, IconButton} from './Button';
import ButtonGroup from './ButtonGroup';
import Card from './Card';
import {ConditionalWrapper} from './ConditionalWrapper';
import {DateTimePicker} from './DateTimePicker';
import DateTimeRangePicker, {DateTimeRange} from './DateTimeRangePicker';
import Dropdown from './Dropdown';
import EmptyState from './EmptyState';
import GrpcMetadataContext, {GrpcMetadataProvider, useGrpcMetadata} from './GrpcMetadataContext';
import Input from './Input';
import {KeyDownProvider, useKeyDown} from './KeyDownContext';
import Modal from './Modal';
import {NoDataPrompt} from './NoDataPrompt';
import ParcaContext from './ParcaContext';
import Pill, {PillVariant} from './Pill';
import RefreshButton from './RefreshButton';
import ResponsiveSvg from './ResponsiveSvg';
import Select, {type SelectElement, type SelectItem} from './Select';
import FlameGraphSkeleton, {FlameActionButtonPlaceholder} from './Skeletons/FlamegraphSkeleton';
import MetricsGraphSkeleton from './Skeletons/MetricsGraphSkeleton';
import SandwichFlameGraphSkeleton from './Skeletons/SandwichFlameGraphSkeleton';
import SourceSkeleton from './Skeletons/SourceSkeleton';
import TableSkeleton, {TableActionButtonPlaceholder} from './Skeletons/TableSkeleton';
import Spinner from './Spinner';
import Tab from './Tab';
import Table from './Table';
import TextWithTooltip from './TextWithTooltip';
import UserPreferences, {UserPreferencesModal} from './UserPreferences';

export type {PillVariant, SelectElement, SelectItem};

export * from './CopyToClipboard';
export * from './ParcaContext';
export * from './hooks/URLState';
export * from './DividerWithLabel';

export {
  Button,
  ButtonGroup,
  Card,
  ConditionalWrapper,
  DateTimePicker,
  DateTimeRange,
  DateTimeRangePicker,
  Dropdown,
  FlameGraphSkeleton,
  GrpcMetadataContext,
  GrpcMetadataProvider,
  FlameActionButtonPlaceholder,
  SandwichFlameGraphSkeleton,
  IconButton,
  Input,
  KeyDownProvider,
  MetricsGraphSkeleton,
  Modal,
  NoDataPrompt,
  ParcaContext,
  Pill,
  ResponsiveSvg,
  RefreshButton,
  Select,
  SourceSkeleton,
  Spinner,
  Tab,
  Table,
  TableActionButtonPlaceholder,
  TableSkeleton,
  TextWithTooltip,
  EmptyState,
  useGrpcMetadata,
  useKeyDown,
  UserPreferences,
  UserPreferencesModal,
};
