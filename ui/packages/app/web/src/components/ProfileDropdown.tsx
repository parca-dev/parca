import React from 'react'
import { Button, ButtonGroup, Dropdown, OverlayTrigger, Tooltip } from 'react-bootstrap'

interface ProfileDropdownProp {
  disabled: boolean
  setProfileName: (profileName: string) => void
  items: string[]
  selected: string
  comparing: boolean
}

const ProfileDropdown = ({ disabled, setProfileName, items, selected, comparing }: ProfileDropdownProp): JSX.Element => {
  const defaultItem = 'Select Profile'
  const disabledStyle = disabled ? { pointerEvents: 'none' } : {}

  const dropdown = (
    <Dropdown>
      <Dropdown.Toggle
        disabled={disabled}
        variant="outline-secondary"
        style={{ ...(disabledStyle as React.CSSProperties), border: 0 }}
      >
        {selected === '' ? defaultItem : selected}
      </Dropdown.Toggle>
      <Dropdown.Menu>
        <Dropdown.Item onSelect={() => setProfileName('')}>
          {defaultItem}
        </Dropdown.Item>
        {items.map((v, i) => (
          <Dropdown.Item key={i} onSelect={() => setProfileName(v)} active={selected === v}>
            {v}
          </Dropdown.Item>
        ))}
      </Dropdown.Menu>
    </Dropdown>
  )

  const buttons = (
    <ButtonGroup>
      {items.map((v: string, i: number) => (
        <Button variant="light" key={i} onClick={() => setProfileName(v)} active={selected === v}>{v}</Button>
      ))}
    </ButtonGroup>
  )

  return (
    <>
      {disabled
        ? (
        <OverlayTrigger
          placement="bottom"
          overlay={
            <Tooltip id="enforced-profile-name">
              Comparing can only be done with profiles of the same type.
            </Tooltip>
          }>

          <span className="d-inline-block">
            {dropdown}
          </span>
        </OverlayTrigger>
          )
        : (
        <>
          {comparing
            ? (
                dropdown
              )
            : (
            <>
              <div className="d-none d-xl-block">{buttons}</div>
              <div className="d-xl-none">{dropdown}</div>
            </>
              )}
        </>
          )}
    </>
  )
}

export default ProfileDropdown
