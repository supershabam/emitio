import * as React from "react";
import * as ReactDOM from "react-dom";

import MuiThemeProvider from "material-ui/styles/MuiThemeProvider";
import AppBar from "material-ui/AppBar";
import RaisedButton from "material-ui/RaisedButton";
import Card from "material-ui/Card";
import "rxjs/add/operator/do";
import "rxjs/add/operator/filter";
import "rxjs/add/operator/delay";
import "rxjs/add/operator/mergeMap";
import "rxjs/add/operator/map";
import "rxjs/add/operator/mapTo";
import { ajax } from "rxjs/observable/dom/ajax";
import { webSocket } from "rxjs/observable/dom/websocket";

import { createStore, applyMiddleware, compose } from "redux";
import { createEpicMiddleware, combineEpics } from "redux-observable";
import { Provider } from "react-redux";
import Editor from "./containers/Editor";
import Rows from "./containers/Rows";
import Heatmap from "./containers/Heatmap";

const reducer = (prev, action) => {
  switch (action.type) {
    case "SUBMIT":
      return { ...prev, ...{ rows: [] } };
    case "EDITOR_CHANGE":
      return { ...prev, ...{ value: action.value } };
    case "READ_ROWS_REPLY":
      let rows = (action.reply.rows || []).map(row => {
        return JSON.parse(row);
      });
      return { ...prev, ...{ rows: prev.rows.concat(rows) } };
    case "FETCH_HEATMAP_FULFILLED":
      return { ...prev, ...{ heatmap: action.response } };
    default:
      return prev;
  }
};
const submitEpic = (action$, store) => {
  return action$.filter(action => action.type === "SUBMIT").mergeMap(action => {
    let ws = webSocket("ws://edge.emit.io:9009/");
    ws.next(JSON.stringify({ javascript: store.getState()["value"] }));
    return ws.map(next => {
      return { type: "READ_ROWS_REPLY", reply: next };
    });
  });
};
const fetchHeatmapEpic = (action$, store) => {
  return action$
    .filter(action => action.type === "FETCH_HEATMAP")
    .mergeMap(action =>
      ajax.getJSON("http://localhost:8080/").map(response => ({
        type: "FETCH_HEATMAP_FULFILLED",
        response: response
      }))
    );
};

const epic = combineEpics(submitEpic, fetchHeatmapEpic);
const init = {
  value: `function transform(acc, line) {
  return [acc, [line]]
}`,
  rows: [],
  heatmap: {
    histograms: []
  }
};
const store = createStore(
  reducer,
  init,
  applyMiddleware(createEpicMiddleware(epic))
);
store.dispatch({ type: "FETCH_HEATMAP" });
// store.subscribe(() => console.log("store", store.getState()));

const App = () => (
  <MuiThemeProvider>
    <div>
      <AppBar title="emitio" />
      <p>javascript reducer</p>
      <Card>
        <Editor />
      </Card>
      <br />
      <RaisedButton
        primary={true}
        onClick={() => store.dispatch({ type: "SUBMIT" })}
      >
        Submit
      </RaisedButton>
      <hr />
      <Rows />
    </div>
  </MuiThemeProvider>
);

ReactDOM.render(
  <Provider store={store}>
    <Heatmap heatmap={init.heatmap} />
  </Provider>,
  document.getElementById("root")
);
