import Head from 'next/head'
import Link from 'next/link'
import { Navbar } from 'react-bootstrap'
import { NextRouter, withRouter } from 'next/router'
import { Parca } from '@parca/icons'

interface HeaderProps {
  router: NextRouter
}

const Header = (_: HeaderProps): JSX.Element => {
  return (
    <>
      <Head>
        <title>Parca</title>
        <link rel='icon' href='/favicon.svg' />
      </Head>
      <Navbar
        collapseOnSelect
        expand='lg'
        bg='light'
        variant='light'
        style={{ borderBottom: '1px solid #E4E8F0' }}
      >
        <Link href='/' passHref>
          <Navbar.Brand style={{ marginLeft: 56 }}>
            <Parca width={200} height={34} />
          </Navbar.Brand>
        </Link>
        <Navbar.Toggle aria-controls='responsive-navbar-nav' />
      </Navbar>
    </>
  )
}

export default withRouter(Header)
