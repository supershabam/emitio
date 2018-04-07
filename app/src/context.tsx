import * as React from "react";
import { Action } from "./actions";
import { State } from "./state";
import { of, Observable, Subject, Subscription } from "rxjs";
import { map, flatMap, publishBehavior, refCount } from "rxjs/operators";

export const { Provider, Consumer } = React.createContext({});

type AffectorFn = <S, A>(
  s: S,
  a: A,
  s$: Observable<S>,
  a$: Observable<A>
) => [S, Observable<A>];

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

export const createStore = (affector: AffectorFn | any, init: State) => {
  const action$$ = new Subject<Observable<Action>>();
  const state$ = affect(affector, init, action$$);
  const dispatch = (action: Action) => {
    action$$.next(of(action));
  };
  return {
    state$,
    dispatch
  };
};

export const connect = (
  WrappedComponent: any,
  mapState$ToProps?: any,
  mapDispatchToProps?: any
) => {
  interface ConnectProps {
    store: {
      dispatch: (a: Action) => null;
      state$: Observable<State>;
    };
  }
  class Connect extends React.Component<ConnectProps> {
    static WrappedComponent = WrappedComponent;
    subs: Array<Subscription>;
    constructor(props: ConnectProps) {
      super(props);
      this.state = {};
      this.subs = [];
    }
    componentDidMount() {
      for (let key in mapState$ToProps) {
        this.subs = this.subs.concat([
          mapState$ToProps[key](this.props.store.state$).subscribe(value => {
            this.setState(prevState => {
              return { ...prevState, [key]: value };
            });
          })
        ]);
      }
    }
    componentWillUnmount() {
      for (let sub of this.subs) {
        sub.unsubscribe();
      }
    }
    render() {
      return (
        <WrappedComponent
          state$={this.props.store.state$}
          dispatch={this.props.store.dispatch}
          {...(this.props, this.state)}
        />
      );
    }
  }
  return props => {
    return <Consumer>{store => <Connect store={store} {...props} />}</Consumer>;
  };
};
