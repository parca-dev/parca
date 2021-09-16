import type { AppProps } from 'next/app'
import { Container } from 'react-bootstrap'
import 'react-dates/lib/css/_datepicker.css'
import 'tailwindcss/tailwind.css'
import '../style/file-input.css'
import '../style/globals.scss'
import '../style/metrics.css'
import '../style/profile.css'
import '../style/sidenav.css'
import './App.scss'
import Header from './layouts/Header'

const MyApp = ({ Component, pageProps }: AppProps) => {
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
