import type { AppProps } from 'next/app'
import { Container } from 'react-bootstrap'
import 'react-dates/lib/css/_datepicker.css'
import { StoreProvider, useCreateStore } from 'store'
import 'tailwindcss/tailwind.css'
import '../style/file-input.css'
import '../style/globals.scss'
import '../style/metrics.css'
import '../style/profile.css'
import '../style/sidenav.css'
import './App.scss'
import Header from './layouts/Header'

const App = ({ Component, pageProps }) => {
  const { persistedState } = pageProps
  // this is only point where persisted state can come in. it's either from:
  // - cookies headers (server)
  // - window.__NEXT_DATA__ (client)
  const createStore = useCreateStore(persistedState?.state)

  return (
    <StoreProvider createStore={createStore}>
        <Header />
        <Container fluid>
          <Component {...pageProps} />
        </Container>
    </StoreProvider>
  )
}

export default App
