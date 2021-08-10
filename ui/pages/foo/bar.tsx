import Link from "next/link";
import styles from "../../styles/Home.module.css";

function Bar(): JSX.Element {
  return (
      <div className={styles.container}>
        <main className={styles.main}>
          <h1 className={styles.title}>
            Bar
          </h1>

          <p className={styles.description}>
            Get started by editing{' '}
            <code className={styles.code}>ages/foo/bar.ts</code>
          </p>

          <p>
            Check out <Link href="/">the homepage</Link>.
          </p>
        </main>
      </div>
  );
}

export default Bar;
