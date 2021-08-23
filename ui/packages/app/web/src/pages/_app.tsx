// import App from "next/app";
import './App.scss'
import type { AppProps } from 'next/app'
import Header from './layouts/Header'
import Sidenav from './layouts/Sidenav'
import { Col, Container, Row } from 'react-bootstrap'
import '../style/sidenav.css'
import '../style/profile.css'
import '../style/metrics.css'
import 'react-dates/lib/css/_datepicker.css'
// import { startsWith } from '../libs/utils'
import '../style/file-input.css'

const MyApp = ({ Component, pageProps }: AppProps): JSX.Element => {
  return (
    <>
      <Header />
      <Container fluid>
        <Row>
          <Col xs={1} id='sidebar-wrapper'>
            <Sidenav />
          </Col>
          <Col xs={12} id='page-content-wrapper' style={{ paddingLeft: 79 }}>
            <Component {...pageProps} />
          </Col>
        </Row>
      </Container>
    </>
  )
  // }

  // return (
  //   <>
  //     <Header apiEndpoint={apiEndpoint} />
  //     <Container fluid>
  //       <Component {...pageProps} />
  //     </Container>
  //   </>
  // )
}

// Only uncomment this method if you have blocking data requirements for
// every single page in your application. This disables the ability to
// perform automatic static optimization, causing every page in your app to
// be server-side rendered.
//
// MyApp.getInitialProps = async (appContext: AppContext) => {
//   // calls page's `getInitialProps` and fills `appProps.pageProps`
//   const appProps = await App.getInitialProps(appContext);

//   return { ...appProps }
// }

export default MyApp
