import { Nav } from 'react-bootstrap'
import { NextRouter, withRouter } from 'next/router'
import Link from 'next/link'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faNetworkWired } from '@fortawesome/free-solid-svg-icons'
import { startsWith } from '../../libs/utils'

interface SidenavProps {
  router: NextRouter
}

const Sidenav = ({ router }: SidenavProps): JSX.Element => {
  const pathname = router.pathname

  return (
    <>
      <Nav className='d-none d-md-block bg-light sidebar'>
        <div className='sidebar-sticky'></div>
        <Nav.Item className={startsWith(pathname, '/profiles') ? 'active' : 'inactive'}>
          {/* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */}
          <Link href={'/profiles'}>
            <Nav.Link as='span'>
              <FontAwesomeIcon icon={faNetworkWired} />
            </Nav.Link>
          </Link>
        </Nav.Item>
      </Nav>
    </>
  )
}

export default withRouter(Sidenav)
