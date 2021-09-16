import dynamic from 'next/dynamic'
import { Container } from 'react-bootstrap'
import 'react-dates/lib/css/_datepicker.css'
import { StoreProvider, useCreateStore } from 'store'
import 'tailwindcss/tailwind.css'
import '../style/file-input.css'
import '../style/metrics.css'
import '../style/profile.css'
import '../style/sidenav.css'
import './App.scss'
import Header from './layouts/Header'
import ThemeProvider from './layouts/ThemeProvider'

const NoSSR = dynamic(() => import('../components/NoSSR'), { ssr: false })

const App = ({ Component, pageProps }) => {
  const createStore = useCreateStore()

  return (
    <NoSSR>
      <StoreProvider createStore={createStore}>
        <ThemeProvider>
          <Header />
          <Container fluid>
            <Component {...pageProps} />
          </Container>
        </ThemeProvider>
      </StoreProvider>
    </NoSSR>
  )
}

export default App
