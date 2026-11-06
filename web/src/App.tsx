import { Layout } from './components/Layout';
import { I18nProvider } from './i18n/i18n';
import { useRoute } from './lib/router';
import { Find } from './pages/Find';
import { Home } from './pages/Home';
import { About, NotFound, Privacy } from './pages/Simple';
import { Trip } from './pages/Trip';

function Routes() {
  const route = useRoute();
  switch (route.name) {
    case 'home':
      return <Home />;
    case 'find':
      return <Find />;
    case 'train':
      return <Trip key={route.train} train={route.train} />;
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
