import Link from 'next/link'
import styles from '../../styles/Home.module.css'

function Foo (): JSX.Element {
  return (
      <div className={styles.container}>
        <main className={styles.main}>
          <h1 className={styles.title}>
            Foo
          </h1>

          <p className={styles.description}>
            Get started by editing{' '}
            <code className={styles.code}>pages/foo/index.tsx</code>
          </p>

          <p>
            Check out <Link href="/foo/bar">bar</Link>.
          </p>
        </main>
      </div>
  )
}

export default Foo
