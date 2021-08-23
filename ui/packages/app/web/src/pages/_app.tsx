import './App.scss'
import type { AppProps } from 'next/app'
import Header from './layouts/Header'
import { Container } from 'react-bootstrap'
import '../style/sidenav.css'
import '../style/profile.css'
import '../style/metrics.css'
import 'react-dates/lib/css/_datepicker.css'
import '../style/file-input.css'

const MyApp = ({ Component, pageProps }: AppProps): JSX.Element => {
  return (
    <>
      <Header />
      <Container fluid>
        <Component {...pageProps} />
      </Container>
    </>
  )
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
