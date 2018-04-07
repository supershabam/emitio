import * as React from "react";
import * as ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Link } from "react-router-dom";
import { State } from "./state";
import { affector } from "./affector";
import { Provider, createStore } from "./context";
import App from "./containers/App";

const init: State = {
  drawer: {
    open: false
  }
};

const { state$, dispatch } = createStore(affector, init);

ReactDOM.render(
  <Provider value={{ dispatch, state$ }}>
    <App />
  </Provider>,
  document.getElementById("app")
);
