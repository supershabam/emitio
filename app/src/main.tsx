import * as React from "react";
import * as ReactDOM from "react-dom";
import { connect, createContext } from "reglaze";
import { BrowserRouter as Router } from "react-router-dom";
import { State } from "./state";
import { affector } from "./affector";
import App from "./containers/App";
import { Observable } from "indefinite-observable";

const init: State = {
  user: null,
  service: {
    refreshing: false,
    services: ["prod-edge", "customer-api"]
  }
};

const { Provider } = createContext("main", affector, init);

ReactDOM.render(
  <Router>
    <Provider>
      <App />
    </Provider>
  </Router>,
  document.getElementById("app")
);
