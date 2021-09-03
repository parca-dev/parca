import Navbar from 'components/ui/Navbar'
import Head from 'next/head'
import { withRouter } from 'next/router'

const Header = ({}) => {
  return (
    <>
      <Head>
        <title>Parca</title>
        <link rel='icon' href='/favicon.svg' />
      </Head>
      <Navbar />
    </>
  )
}

export default withRouter(Header)
