import { Disclosure } from '@headlessui/react'
import { MenuIcon, XIcon } from '@heroicons/react/outline'
import { Parca, ParcaSmall } from '@parca/icons'
import cx from 'classnames'

const links = [{ name: 'Profiles', href: '/', current: true }]

const Navbar = () => {
  return (
    <Disclosure as='nav' className='bg-gray-800 relative z-10'>
      {({ open }) => (
        <>
          <div className='mx-auto px-3 sm:px-6 lg:px-8 xl:px-0'>
            <div className='relative flex items-center justify-between h-16'>
              <div className='absolute inset-y-0 left-0 flex items-center sm:hidden'>
                {/* mobile menu button */}
                <Disclosure.Button className='inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white'>
                  <span className='sr-only'>Open main menu</span>
                  {open ? (
                    <XIcon className='block h-6 w-6' aria-hidden='true' />
                  ) : (
                    <MenuIcon className='block h-6 w-6' aria-hidden='true' />
                  )}
                </Disclosure.Button>
              </div>
              <div className='flex-1 flex items-center justify-center sm:items-stretch sm:justify-start'>
                <div className='flex-shrink-0 flex items-center'>
                  {/* image for small screens: */}
                  <div
                    style={{ padding: '5px' }}
                    className='block lg:hidden h-8 w-auto rounded-full'
                  >
                    <ParcaSmall
                      style={{ height: '100%', width: '100%', filter: 'invert(1)' }}
                      className='block lg:hidden h-8 w-auto'
                    />
                  </div>
                  {/* image for larger screens: */}
                  <Parca
                    height={32}
                    style={{ filter: 'invert(1)', transform: 'translateY(5px)' }}
                    className='hidden lg:block h-8 w-auto'
                  />
                </div>
                <div className='hidden sm:block sm:ml-6'>
                  <div className='flex space-x-4'>
                    {links.map(item => (
                      <a
                        key={item.name}
                        href={item.href}
                        className={cx(
                          item.current
                            ? 'bg-gray-900 text-white'
                            : 'text-gray-300 hover:bg-gray-700 hover:text-white',
                          'px-3 py-2 rounded-md text-sm font-medium'
                        )}
                        aria-current={item.current ? 'page' : undefined}
                      >
                        {item.name}
                      </a>
                    ))}
                  </div>
                </div>
              </div>
              {/* placeholder for right menu drop down */}
              {/* <div className='absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0'>
                <Dropdown
                  text=''
                  element={
                    <img
                      className='h-8 w-8 rounded-full'
                      src='data:image/png;base64,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FeHNUZuHu232SHhykGE7OaqaUF1epP7gBCdOLjK3PEVmM1LnuO3f/hLtT7zB7rlt2pNV7EfuYDrtsH11i910xOthm8MrNS51hEagKEqhn3mqAdQjhdEQ6PEiKz2Yvf2CODJUYsVWKTQ+c5r0hbMEmYNI8OLHNsYBpX7HIdMeNELhDLvDkun3HGHu7z2G6+UopTBaoxjhU9jvpDjv8V7AQeXxB5BHLEVuKQYF7Y1rtDsjXOm5tN3jtqUUbETuIQw1XmBUglEem3uSYOzOWe8xQaBRSvBeWN8c8umFKpV//3Oc+Ud/Ti0LUcmBneHH+D32occJGedpUHJlDsxDi8TDFJ95nAMHOP3TUsIzrlrZTSmzEu8cTVOyMxiRjkqiMKC3V9DdGVD1ml5mgZAwVhyKFX03bqW7JhU7A8+VocfUKgFaCaES8qzHcjTixKcexu4Nef7LL1Nc2qWGQaIYNa7/OIGRp0+f8Ogi4W88SP3UEcpuhj/gumP5+o568+8CgVJCoIQUsNYzchkqyymKkkqSMNUMuDXMiELPelrivaAkYCdQpNbRS6GTeUoPaekxWo0p7Ajwg4JWvUqxN2Rido6Hv/hZNp+/yc5rF2lfWEP1R1gcgiaar3H33/koj3/hQywdWeHm7pBd78kP/o2ixB+E7HGAPQBYJYJS42tJy5ILV7fY66eUdqyn6xMV8sKiI3AH7eqAvdH48SQU6JXjtwQTCKYeQYHHOhAcBsXe+W2CskLdtpD3HyN55HHqvR6Tkx0OtUpa8zVWbl9gtt6iIMWOhkxG4/e99ghyHJ4Dx+VgMZiDmxgbFg6Po93tsL7RxpYOkbFNE4cRZZmhtGKmYoijgNCMEcgdvCNqMx5iEfi/8IQxsW22zOAAAAAHdEVYdEF1dGhvcgCprsxIAAAACnRFWHRDb3B5cmlnaHQArA/MOgAAAA50RVh0Q3JlYXRpb24gdGltZQA19w8JAAAAJXRFWHRkYXRlOmNyZWF0ZQAyMDE4LTA0LTE1VDAxOjM1OjQ4LTA0OjAwm24ibwAAACV0RVh0ZGF0ZTptb2RpZnkAMjAxOC0wNC0xNVQwMTozNTo0OC0wNDowMOozmtMAAAAMdEVYdERlc2NyaXB0aW9uABMJISMAAAALdEVYdERpc2NsYWltZXIAt8C0jwAAAE10RVh0c29mdHdhcmUASW1hZ2VNYWdpY2sgNi44LjktOSBRMTYgeDg2XzY0IDIwMTctMDUtMjYgaHR0cDovL3d3dy5pbWFnZW1hZ2ljay5vcmcpoK4TAAAAB3RFWHRTb3VyY2UA9f+D6wAAABh0RVh0VGh1bWI6OkRvY3VtZW50OjpQYWdlcwAxp/+7LwAAABh0RVh0VGh1bWI6OkltYWdlOjpIZWlnaHQAMTQwG/1uNAAAABd0RVh0VGh1bWI6OkltYWdlOjpXaWR0aAAxNDJmAl9FAAAAGXRFWHRUaHVtYjo6TWltZXR5cGUAaW1hZ2UvcG5nP7JWTgAAABd0RVh0VGh1bWI6Ok1UaW1lADE1MjM3NzA1NDiLFj0zAAAAE3RFWHRUaHVtYjo6U2l6ZQA0My42S0JCY6rIbAAAACJ0RVh0VGh1bWI6OlVSSQAvdG1wLy9vcmlnaW5hbC9MWlZZUVswXSuG7tcAAAAGdEVYdFRpdGxlAKju0icAAAAIdEVYdFdhcm5pbmcAwBvmhwAAAABJRU5ErkJggg=='
                      alt=''
                    />
                  }
                >
                  <Dropdown.Item onSelect={() => {}}>Sign out</Dropdown.Item>
                </Dropdown>
              </div> */}
            </div>
          </div>

          <Disclosure.Panel className='sm:hidden'>
            <div className='px-2 pt-2 pb-3 space-y-1'>
              {links.map(item => (
                <a
                  key={item.name}
                  href={item.href}
                  className={cx(
                    item.current
                      ? 'bg-gray-900 text-white'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white',
                    'block px-3 py-2 rounded-md text-base font-medium'
                  )}
                  aria-current={item.current ? 'page' : undefined}
                >
                  {item.name}
                </a>
              ))}
            </div>
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  )
}

export default Navbar
