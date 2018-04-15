import * as React from "react";
import * as ReactDOM from "react-dom";
import { connect, createContext } from "reglaze";
import { BrowserRouter as Router } from "react-router-dom";
import { State } from "./state";
import { affector } from "./affector";
import App from "./containers/App";
import { of } from "rxjs";
import { Action } from "./actions";

const init: State = {
  user: null,
  service: {
    refreshing: false,
    services: []
  }
};

const { Provider } = createContext(
  "main",
  affector,
  init,
  of<Action>({
    kind: "LoginRequest",
    username: "supershabam",
    password: "poop"
  })
);

ReactDOM.render(
  <Router>
    <Provider>
      <App />
    </Provider>
  </Router>,
  document.getElementById("app")
);
