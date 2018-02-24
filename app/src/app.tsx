import * as React from "react";
import * as ReactDOM from "react-dom";
import MuiThemeProvider from "material-ui/styles/MuiThemeProvider";
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
import App from "./containers/App";
import Service from "./containers/Service";

const reducer = (prev, action) => {
  console.log(action);
  switch (action.type) {
    case "SET_ACC":
      return { ...prev, ...{ acc: action.acc } };
    case "CHANGE_TAB":
      return { ...prev, ...{ tab: action.value } };
    case "READ_NODE":
      return { ...prev, ...{ rows: [], lastAcc: "" } };
    case "EDITOR_CHANGE":
      return { ...prev, ...{ value: action.value } };
    case "READ_NODE_REPLY":
      let rows = (action.reply.rows || []).map(row => {
        return row;
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

const js = `var SYSLOG_LINE_REGEX = new RegExp(
  [
    /(<[0-9]+>)?/, // 1 - optional priority
    /([a-z]{3})\\s+/, // 2 - month
    /([0-9]{1,2})\\s+/, // 3 - date
    /([0-9]{2}):/, // 4 - hours
    /([0-9]{2}):/, // 5 - minutes
    /([0-9]{2})/, // 6 - seconds
    /(\\s+[\\w.-]+)?\\s+/, // 7 - host
    /([\\w\\-().0-9/]+)/, // 8 - process
    /(?:\\[([a-z0-9-.]+)\\])?:/, // 9 - optional pid
    /(.+)/ // 10  message
  ]
    .map(function(regex) {
      return regex.source;
    })
    .join(""),
  "i"
);

var FACILITY = [
  "kern",
  "user",
  "mail",
  "daemon",
  "auth",
  "syslog",
  "lpr",
  "news",
  "uucp",
  "cron",
  "authpriv",
  "ftp",
  "ntp",
  "logaudit",
  "logalert",
  "clock",
  "local0",
  "local1",
  "local2",
  "local3",
  "local4",
  "local5",
  "local6",
  "local7"
];

var SEVERITY = [
  "emerg",
  "alert",
  "crit",
  "err",
  "warning",
  "notice",
  "info",
  "debug"
];

var MONTHS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec"
];

function syslogParse(log) {
  var parts = SYSLOG_LINE_REGEX.exec(log.trim());
  if (!parts) {
    return {};
  }

  var priority = Number((parts[1] || "").replace(/[^0-9]/g, ""));
  var facilityCode = priority >> 3;
  var facility = FACILITY[facilityCode];
  var severityCode = priority & 7;
  var severity = SEVERITY[severityCode];

  var month = MONTHS.indexOf(parts[2]);
  var date = Number(parts[3]);
  var hours = Number(parts[4]);
  var minutes = Number(parts[5]);
  var seconds = Number(parts[6]);

  var time = new Date();
  time.setMonth(month);
  time.setDate(date);
  time.setHours(hours);
  time.setMinutes(minutes);
  time.setSeconds(seconds);

  var host = (parts[7] || "").trim();
  var processName = parts[8];
  var pid = Number(parts[9]);

  var message = parts[10].trim();

  var result = {
    priority: priority,
    facilityCode: facilityCode,
    facility: facility,
    severityCode: severityCode,
    severity: severity,
    time: time,
    host: host,
    process: processName,
    message: message
  };

  if (pid) {
    result.pid = pid;
  }

  return result;
}

function transform(acc, lines) {
  var a = JSON.parse(acc);
  var p = a.process || "";
  return [
    acc,
    lines
      .map(function(line) {
        try {
          var l = JSON.parse(line);
          var msg = syslogParse(l.r);
          l.syslog = msg;
          return l;
        } catch (e) {
          return "";
        }
      })
      .filter(function(line) {
        return line !== "";
      })
      .filter(function(line) {
        return line.syslog.process === p;
      })
      .map(function(line) {
        return JSON.stringify(line.syslog);
      })
  ];
}

`;

const epic = combineEpics(fetchHeatmapEpic, fetchNodesEpic, readNodeEpic);
const init = {
  tab: "reducer",
  acc: `{"process":"nginx"}`,
  lastAcc: `{"process":"nginx"}`,
  nodes: [],
  value: js,
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
store.dispatch({ type: "FETCH_HEATMAP" });
store.subscribe(() => console.log("store", store.getState()));

ReactDOM.render(
  <Provider store={store}>
    <MuiThemeProvider>
      <Service />
    </MuiThemeProvider>
  </Provider>,
  document.getElementById("root")
);
