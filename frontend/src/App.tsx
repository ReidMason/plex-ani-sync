import { BrowserRouter as Router } from 'react-router-dom';
import Routes from './Routes';

function App() {
  return (
    <Router>
      <Routes />
      <p className="absolute m-4 bottom-0 text-3xl font-bold text-gray-500">BETA</p>
    </Router>
  );
}

export default App;
