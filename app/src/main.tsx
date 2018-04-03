import * as React from "react";
import * as ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Link } from "react-router-dom";
import {
  Observable,
  Subject,
  ReplaySubject,
  from,
  of,
  range,
  empty
} from "rxjs";
import {
  map,
  catchError,
  tap,
  flatMap,
  startWith,
  scan,
  delay,
  filter,
  takeUntil
} from "rxjs/operators";
import { ajax } from "rxjs/ajax";

interface LoginRequest {
  kind: "LoginRequest";
  username: string;
  password: string;
}

interface LoginSuccess {
  kind: "LoginSuccess";
  username: string;
}

interface LoginError {
  kind: "LoginError";
  error: string;
}

interface Logout {
  kind: "logout";
}

interface State {
  username?: string;
}

type Action = LoginRequest | LoginSuccess | LoginError | Logout;

const reducer = (state: State, action: Action): State => {
  console.log("before", state, action);
  switch (action.kind) {
    case "LoginRequest":
      return { ...state, username: null };
    case "LoginSuccess":
      return { ...state, username: action.username };
    case "logout":
      return { ...state, username: null };
  }
  return state;
};

const affector = (action: Action, state: State): Observable<Action> => {
  switch (action.kind) {
    case "LoginRequest":
      if (action.password == "pie") {
        return of<Action>({
          kind: "LoginSuccess",
          username: action.username
        }).pipe(delay(250));
      }
      return of<Action>({
        kind: "LoginError",
        error: "what even is the pie"
      }).pipe(delay(250));
  }
  return empty();
};

const action$ = new Subject<Observable<Action>>();
const createStore = (init: State) => {
  return action$.pipe(
    flatMap(action => action),
    // we need to delay the result of affector or else the "affected" actions will
    // execute before the action that triggered them
    tap(action => action$.next(affector(action).pipe(delay(0)))),
    scan(reducer, init)
  );
};

const store = createStore({});

store.subscribe(console.log);

action$.next(
  of<Action>({ kind: "LoginRequest", username: "dingleberry", password: "pie" })
);
action$.next(of<Action>({ kind: "logout" }).pipe(delay(1500)));
