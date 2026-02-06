import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
        <div className={styles.heroTopContainer}>
            <div className={styles.heroTextContainer}>
                <Heading as="h1" className={`hero__title ${styles.heroTitle}`}>
                    {siteConfig.title}
                </Heading>
                <p className={`hero__subtitle ${styles.heroSubtitle}`}>{siteConfig.tagline}</p>
                <div className={styles.buttons}>
                <Link className="button button--secondary button--lg" to="/docs/getting-started/introduction">
                    Get started
                </Link>
            </div>
          </div>
          <img src="img/social_card.png" alt="Expo Open OTA" className={styles.imgHeader} />
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description={siteConfig.tagline}
      >
      <HomepageHeader />
      <main>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
