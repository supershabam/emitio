import * as React from "react";
import * as ReactDOM from "react-dom";
import * as Reglaze from "reglaze";
import { BrowserRouter as Router } from "react-router-dom";
import { State } from "./state";
import { affector } from "./affector";
import { Provider, createStore } from "./context";
import App from "./containers/App";

const init: State = {
  user: null
};

const { state$, dispatch } = createStore(affector, init);

ReactDOM.render(
  <Router>
    <Provider value={{ dispatch, state$ }}>
      <App />
    </Provider>
  </Router>,
  document.getElementById("app")
);
