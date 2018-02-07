import * as React from "react";
import * as ReactDOM from "react-dom";

import MuiThemeProvider from "material-ui/styles/MuiThemeProvider";
import AppBar from "material-ui/AppBar";
import RaisedButton from "material-ui/RaisedButton";
import Card from "material-ui/Card";
import "rxjs/add/operator/do";
import "rxjs/add/operator/catch";
import "rxjs/add/operator/filter";
import "rxjs/add/operator/delay";
import "rxjs/add/operator/mergeMap";
import "rxjs/add/operator/map";
import "rxjs/add/operator/switchMap";
import "rxjs/add/operator/mapTo";
import { ajax } from "rxjs/observable/dom/ajax";
import { webSocket } from "rxjs/observable/dom/websocket";
import * as Rx from "rxjs";

import { createStore, applyMiddleware, compose } from "redux";
import { createEpicMiddleware, combineEpics } from "redux-observable";
import { Provider } from "react-redux";
import Editor from "./containers/Editor";
import Rows from "./containers/Rows";
import Heatmap from "./containers/Heatmap";
import NodeCount from "./containers/NodeCount";

const reducer = (prev, action) => {
  switch (action.type) {
    case "READ_NODE":
      return { ...prev, ...{ rows: [], lastAcc: "" } };
    case "EDITOR_CHANGE":
      return { ...prev, ...{ value: action.value } };
    case "READ_NODE_REPLY":
      let rows = (action.reply.rows || []).map(row => {
        return JSON.parse(row);
      });
      return {
        ...prev,
        ...{
          rows: prev.rows.concat(rows),
          lastAcc: action.reply.last_accumulator
        }
      };
    case "FETCH_HEATMAP_FULFILLED":
      return { ...prev, ...{ heatmap: action.response } };
    case "FETCH_NODES_FULFILLED":
      return { ...prev, ...{ nodes: action.response.nodes || [] } };
    default:
      return prev;
  }
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
const fetchNodesEpic = (action$, store) => {
  return action$
    .filter(action => action.type === "FETCH_NODES")
    .mergeMap(action =>
      ajax.getJSON("http://edge.emit.io:9009/v0/nodes").map(response => ({
        type: "FETCH_NODES_FULFILLED",
        response: response
      }))
    );
};
const readNodeEpic = (action$, store) => {
  return action$
    .filter(action => action.type == "READ_NODE")
    .switchMap(action => {
      let ws = webSocket("ws://edge.emit.io:9009/v0/readnode");
      ws.next(JSON.stringify(action.request));
      return ws.catch(() => Rx.Observable.empty()).map(next => {
        return { type: "READ_NODE_REPLY", reply: next };
      });
    });
};

const epic = combineEpics(fetchHeatmapEpic, fetchNodesEpic, readNodeEpic);
const init = {
  lastAcc: "",
  nodes: [],
  value: `function transform(acc, lines) {
  let a = JSON.parse(acc)
  let out = lines
    .map(line => JSON.parse(line))
    .map(line => {
      return line
    })
    .filter(line => true)
    .map(line => JSON.stringify(line))
  a.count = (a.count || 0) + 1
  return [JSON.stringify(a), out]
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
store.dispatch({ type: "FETCH_NODES" });
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
        onClick={() =>
          store.dispatch({
            type: "READ_NODE",
            request: {
              node: store.getState().nodes[0],
              accumulator: "{}",
              input_limit: 10000,
              output_limit: 10000,
              duration_limit: 15,
              javascript: store.getState().value
            }
          })
        }
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
    <div>
      <NodeCount />
      <App />
    </div>
  </Provider>,
  document.getElementById("root")
);
