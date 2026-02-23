import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
    {
        title: '⚙️ Production-ready in 10 minutes',
        description: (
            <>
                No database, no complex setup. Connect your cloud storage — <strong>AWS S3</strong>, <strong>Google Cloud Storage</strong>, or any S3-compatible provider — and you’re live. Handles release channels, branches, and runtime versions out of the box.
            </>
        ),
    },
    {
        title: '🚀 One Command to Publish',
        description: (
            <>
                The <code>eoas</code> CLI automates everything — run <code>npx eoas init</code> to configure your project, and <code>npx eoas publish</code> to push updates from your CI/CD pipeline. No extra scripts, no hassle.
            </>
        ),
    },
    {
        title: '⚡ Fast Asset Delivery',
        description: (
            <>
                Assets served at the edge. Deliver updates via <strong>CloudFront CDN</strong> or <strong>GCS signed URLs</strong> — your users get updates instantly, wherever they are. No public bucket access needed.
            </>
        ),
    },
];


function Feature({title, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
