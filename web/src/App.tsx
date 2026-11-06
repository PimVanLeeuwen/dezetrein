import { Layout } from './components/Layout';
import { I18nProvider } from './i18n/i18n';
import { useRoute } from './lib/router';
import { Home } from './pages/Home';
import { About, NotFound, Privacy, Train } from './pages/Simple';

function Routes() {
  const route = useRoute();
  switch (route.name) {
    case 'home':
      return <Home />;
    case 'train':
      return <Train key={route.train} train={route.train} />;
    case 'privacy':
      return <Privacy />;
    case 'about':
      return <About />;
    default:
      return <NotFound />;
  }
}

export default function App() {
  return (
    <I18nProvider>
      <Layout>
        <Routes />
      </Layout>
    </I18nProvider>
  );
}
