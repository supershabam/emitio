import * as React from "react";
import { withStyles } from "material-ui/styles";
import AppBar from "material-ui/AppBar";
import Toolbar from "material-ui/Toolbar";
import Typography from "material-ui/Typography";
import Button from "material-ui/Button";
import IconButton from "material-ui/IconButton";
import MenuIcon from "material-ui-icons/Menu";

import { Observable } from "rxjs";
import { State } from "../state";
import { map } from "rxjs/operators";
import { connect } from "../context";

const styles = {
  root: {
    flexGrow: 1
  },
  flex: {
    flex: 1
  },
  menuButton: {
    marginLeft: -12,
    marginRight: 20
  }
};

function ButtonAppBar(props) {
  const { classes, dispatch, username } = props;
  return (
    <div className={classes.root}>
      <AppBar position="static">
        <Toolbar>
          <IconButton
            className={classes.menuButton}
            color="inherit"
            aria-label="Menu"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="title" color="inherit" className={classes.flex}>
            greetings {username}
          </Typography>
          {username ? (
            <Button
              onClick={() =>
                dispatch({
                  kind: "Logout"
                })
              }
              color="inherit"
            >
              Logout
            </Button>
          ) : (
            <Button
              onClick={() =>
                dispatch({
                  kind: "LoginRequest",
                  username: "supershabam",
                  password: "pews"
                })
              }
              color="inherit"
            >
              Login
            </Button>
          )}
        </Toolbar>
      </AppBar>
    </div>
  );
}

export default connect(withStyles(styles)(ButtonAppBar), {
  username: (state$: Observable<State>) => {
    return state$.pipe(
      map((state: State) => {
        if (state.user) {
          return state.user.name;
        }
        return "";
      })
    );
  }
});
