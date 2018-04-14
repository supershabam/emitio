import * as React from "react";
import { Route, Switch } from "react-router-dom";
import CssBaseline from "material-ui/CssBaseline";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { connect } from "../context";
import { State } from "../state";
import { Action } from "../actions";
import Home from "./home";
import AppBar from "./AppBar";

const App = props => {
  return (
    <CssBaseline>
      <div>
        <AppBar>emitio</AppBar>
        <Switch>
          <Route path="/" exact component={Home} />
          <Route component={() => <h2>no match</h2>} />
        </Switch>
      </div>
    </CssBaseline>
  );
};

export default App;
