import * as React from "react";
import * as ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Link } from "react-router-dom";
import { scaleLinear } from "d3-scale";
import { axisLeft } from "d3-axis";
import { select } from "d3-selection";
import {
  Observable,
  Observer,
  Subject,
  BehaviorSubject,
  Subscription,
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
  takeUntil,
  refCount,
  publish,
  publishBehavior,
  shareReplay
} from "rxjs/operators";
import { ajax } from "rxjs/ajax";
import Button from "material-ui/Button";
import Drawer from "material-ui/Drawer";

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
  kind: "Logout";
}

interface State {
  drawer: {
    open: boolean;
  };
  username?: string;
}

type Action = LoginRequest | LoginSuccess | LoginError | Logout;

type AffectorFn = <S, A>(
  s: S,
  a: A,
  s$: Observable<S>,
  a$: Observable<A>
) => [S, Observable<A>];

function affector(
  s: State,
  a: Action,
  s$: Observable<State>,
  a$: Observable<Action>
): [State, Observable<Action>] {
  switch (a.kind) {
    case "LoginRequest":
      return [
        s,
        of<Action>({ kind: "LoginSuccess", username: a.username }).pipe(
          delay(1500),
          takeUntil(
            a$.pipe(
              filter(a => {
                switch (a.kind) {
                  case "LoginSuccess":
                  case "LoginRequest":
                  case "Logout":
                    return true;
                }
                return false;
              })
            )
          )
        )
      ];
    case "LoginSuccess":
      return [{ ...s, username: a.username }, empty()];
    case "Logout":
      return [{ ...s, username: null }, empty()];
  }
  return [s, empty()];
}

const affect = (
  affector: AffectorFn | any,
  init: State,
  action$$: Subject<Observable<Action>>
): Observable<State> => {
  const state$ = new Observable(observer => {
    let current = init;
    const action$ = action$$.pipe(flatMap(a$ => a$));
    const sub = action$
      .pipe(
        map(a => {
          return affector(current, a, state$, action$);
        })
      )
      .subscribe(tuple => {
        current = tuple[0];
        observer.next(current);
        action$$.next(tuple[1]);
      });
    return () => {
      sub.unsubscribe();
    };
  });
  return state$.pipe(publishBehavior(init), refCount());
};

const init: State = {
  drawer: {
    open: false
  }
};
const action$$ = new Subject<Observable<Action>>();
const state$ = affect(affector, init, action$$);
const dispatch = (action: Action) => {
  action$$.next(of(action));
};

const { Provider, Consumer } = React.createContext({
  dispatch,
  state$
});

const connect = (
  WrappedComponent: React.Component<any, any>,
  mapState$ToProps?: Function
) => {
  return class Connect extends React.Component {
    static displayName = `Wrapped${WrappedComponent.displayName}`;
    static WrappedComponent = WrappedComponent;
    constructor(props, context) {
      super(props, context);
      this.state = {};
    }
    render() {
      return (
        <Consumer>
          {store => {
            console.log(store);
            return (
              <WrappedComponent
                state$={store.state$}
                dispatch={dispatch}
                {...this.props}
              />
            );
          }}
        </Consumer>
      );
    }
  };
};

const App = props => {
  console.log(props);
  return <h1>hi</h1>;
};

const WApp = connect(App);

ReactDOM.render(
  <Provider value={{ dispatch, state$ }}>
    <WApp />
  </Provider>,
  document.getElementById("app")
);
