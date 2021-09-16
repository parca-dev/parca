// simple wrapper that needs to be imported via next/dynamic.
// this way we CSR the app in dev like in prod (SSG'd).
const NoSSR = ({ children }) => <>{children}</>

export default NoSSR
