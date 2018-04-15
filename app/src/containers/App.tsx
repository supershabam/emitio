import * as React from "react";
import { Route, Switch } from "react-router-dom";
import CssBaseline from "material-ui/CssBaseline";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { connect } from "reglaze";
import { State } from "../state";
import { Action } from "../actions";
import Home from "./home";
import AppBar from "./AppBar";

const ConnectedHome = connect(
  "main",
  Home,
  {
    selected: ($state: Observable<State>): Observable<string> => {
      return $state.pipe(
        map((state: State): string => {
          if (state.service.selected) {
            return state.service.selected;
          }
          return "";
        })
      );
    },
    services: ($state: Observable<State>): Observable<string[]> => {
      return $state.pipe(
        map((state: State): string[] => {
          return state.service.services;
        })
      );
    }
  },
  {
    select: (selected: string): Action => {
      return { kind: "SelectService", selected };
    }
  }
);

const App = props => {
  return (
    <CssBaseline>
      <div>
        <Switch>
          <Route path="/" exact component={ConnectedHome} />
          <Route component={() => <h2>no match</h2>} />
        </Switch>
      </div>
    </CssBaseline>
  );
};

export default App;
