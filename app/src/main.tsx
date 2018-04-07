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
        empty()
        // of<Action>({ kind: "LoginSuccess", username: a.username }).pipe(
        //   delay(1500),
        //   takeUntil(
        //     a$.pipe(
        //       filter(a => {
        //         switch (a.kind) {
        //           case "LoginSuccess":
        //           case "LoginRequest":
        //           case "Logout":
        //             return true;
        //         }
        //         return false;
        //       })
        //     )
        //   )
        // )
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
    console.log("creating observable");
    let current = init;
    const action$ = action$$.pipe(flatMap(a$ => a$));
    observer.next(current);
    const sub = action$
      .pipe(
        tap(a => {
          console.log("running action", a);
        }),
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
      console.log("unsubscribing");
      sub.unsubscribe();
    };
  });
  return state$.pipe(publishBehavior(init), refCount());
};

// const action$$ = new Subject<Observable<Action>>();
// const state$ = affect(affector, {}, action$$);

interface AppProps {
  affector: any;
  init: State;
  action$: Observable<Action>;
}

class Heatmap extends React.Component<null, null> {
  svg: any;

  constructor(props) {
    super(props);
    this.svg = React.createRef();
  }

  componentDidMount() {
    const scale = scaleLinear();
    const axis = axisLeft(scale).tickSize(100);
    select(this.svg.current)
      .append("g")
      .attr("transform", "translate(0,100)")
      .call(axis);

    console.log("scale", scale);
    console.log("axis", axis);
    // const axis = d3.axisLeft(scale);
  }

  render() {
    return (
      <div className="svg-container">
        <svg
          version="1.1"
          viewBox="0 0 100 100"
          preserveAspectRatio="xMinYMin meet"
          className="svg-content"
          transform="translate(0, -100)"
        >
          <svg
            width="100"
            height="100"
            version="1.1"
            className="svg-content"
            ref={this.svg}
          />
        </svg>
      </div>
    );
  }
}

class App extends React.Component<AppProps, State> {
  action$$: Subject<Observable<Action>>;
  state$: Observable<State>;
  sub: Subscription;
  sub2: Subscription;
  constructor(props: AppProps) {
    super(props);

    this.state = { username: null };
  }

  componentDidMount() {
    this.action$$ = new Subject<Observable<Action>>();
    this.state$ = affect(this.props.affector, this.props.init, this.action$$);
    console.log("subscribing to state");
    this.sub = this.state$.subscribe(state => {
      console.log("got ur state", state);
      this.setState(state);
    });
    console.log("subscribing to state again");
    this.sub2 = this.state$.subscribe(state => {
      console.log("got ur state again", state);
    });
    console.log("sending action");
    this.action$$.next(this.props.action$);
  }

  componentWillUnmount() {
    console.log(this.state$);

    console.log("unsubscribing from componetn");
    this.sub.unsubscribe();
    this.sub2.unsubscribe();
    console.log(this.state$);
  }

  public render() {
    return <Heatmap />;
  }
}

ReactDOM.render(
  <App
    affector={affector}
    init={{ username: null }}
    action$={of<Action>({
      kind: "LoginRequest",
      username: "supershabam",
      password: "sekret"
    })}
  />,
  document.getElementById("app")
);

setTimeout(() => {
  ReactDOM.render(<h1>and it gone</h1>, document.getElementById("app"));
}, 500);
