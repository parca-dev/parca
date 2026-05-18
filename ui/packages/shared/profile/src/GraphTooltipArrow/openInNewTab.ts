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

// Open a URL in a new browser tab — robust to held modifier keys.
//
// Why this exists: the tooltip's "freeze" gesture requires the user to be
// holding ⇧ Shift. Browsers route Shift+click on a target="_blank" link to
// "new window" via real-keyboard state at navigation time, so both plain
// <a target="_blank"> clicks and `window.open` fall through to a window
// instead of a tab.
//
// Trick: dispatch a synthetic MouseEvent on a detached anchor with the
// platform's "new-tab" modifier set explicitly (Cmd & Ctrl)
// and Shift cleared. The browser reads modifier flags off the event for its
// routing decision, so this consistently opens in a new tab.
export function openInNewTab(url: string): void {
  const a = document.createElement('a');
  a.href = url;
  a.target = '_blank';
  a.rel = 'noopener noreferrer';

  a.dispatchEvent(
    new MouseEvent('click', {
      bubbles: false,
      cancelable: true,
      view: window,
      button: 0,
      shiftKey: false,
      metaKey: true,
      ctrlKey: true,
    })
  );
}
