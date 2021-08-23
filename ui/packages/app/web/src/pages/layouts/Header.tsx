import Head from 'next/head'
import Link from 'next/link'
import { Navbar } from 'react-bootstrap'
import { NextRouter, withRouter } from 'next/router'

interface HeaderProps {
  router: NextRouter
}

const Header = (_: HeaderProps): JSX.Element => {
  return (
    <>
      <Head>
        <title>Parca</title>
        <link rel='icon' href='/favicon.png' />
      </Head>
      <Navbar
        collapseOnSelect
        expand='lg'
        bg='light'
        variant='light'
        style={{ borderBottom: '1px solid #E4E8F0' }}
      >
        <Link href='/' passHref>
          <Navbar.Brand style={{ marginLeft: 56 }}>{/* TODO(kakkoyun): Parca Logo */}</Navbar.Brand>
        </Link>
        <Navbar.Toggle aria-controls='responsive-navbar-nav' />
      </Navbar>
    </>
  )
}

export default withRouter(Header)
