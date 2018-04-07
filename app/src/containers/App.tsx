import * as React from "react";
import AppBar from "material-ui/AppBar";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { connect } from "../context";
import { State } from "../state";
import { Action } from "../actions";

interface AppProps {
  username?: string;
  dispatch: (a: Action) => null;
}

const App = (props: AppProps) => {
  if (props.username) {
    return (
      <h1 onClick={() => props.dispatch({ kind: "Logout" })}>
        greetings {props.username}
      </h1>
    );
  }
  return (
    <div>
      <h1
        onClick={() =>
          props.dispatch({
            kind: "LoginRequest",
            username: "supershabam",
            password: "things"
          })
        }
      >
        login
      </h1>
    </div>
  );
};

export default connect(App, {
  username: (state$: Observable<State>) => {
    return state$.pipe(map((s: State) => s.username));
  }
});
