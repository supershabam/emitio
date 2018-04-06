import * as React from "react";
import * as ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Link } from "react-router-dom";
import {
  Observable,
  Observer,
  Subject,
  BehaviorSubject,
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
  publishBehavior
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
      sub.unsubscribe();
    };
  });
  return state$;
};

const action$$ = new Subject<Observable<Action>>();
const state$ = affect(affector, {}, action$$);

console.log(state$);
console.log("subscribing");
state$.subscribe(state => {
  console.log(state);
});
console.log("subscribing");
state$.subscribe(state => {
  console.log(state);
});
action$$.next(
  of<Action>({
    kind: "LoginRequest",
    username: "supershabam",
    password: "poop"
  })
);
action$$.next(of<Action>({ kind: "Logout" }));

// const reducer = (state: State, action: Action): State => {
//   console.log("before", state, action);
//   switch (action.kind) {
//     case "LoginRequest":
//       return { ...state, username: null };
//     case "LoginSuccess":
//       return { ...state, username: action.username };
//     case "logout":
//       return { ...state, username: null };
//   }
//   return state;
// };

// const affector = (
//   action$: Observable<Action>,
//   state$: BehaviorSubject<State>
// ): Observable<Action> => {
//   switch (action.kind) {
//     case "LoginRequest":
//       if (action.password == "pie") {
//         return of<Action>({
//           kind: "LoginSuccess",
//           username: action.username
//         }).pipe(delay(250));
//       }
//       return of<Action>({
//         kind: "LoginError",
//         error: "what even is the pie"
//       }).pipe(delay(250));
//   }
//   return empty();
// };

// func affect<S, A>(affector AffectorFn<S,A>, init S) observable<S>

// interface GenericIdentityFn<T> {
//   (arg: T): T;
// }

// function identity<T>(arg: T): T {
//   return arg;
// }

// interface GenericAffectorFn<S, A> {
//   (state: S, action: A, state$: Observable<S>, action$: Observable<A>): [
//     S,
//     Observable<A>
//   ]
// }

// function affect<S,A>(state: S, action: A, state$: Observable<S>, action$: Observable<A>): [
//   S,
//   Observable<A>
// ]

// type AffectorFn {
//   <S, A>(state: S, action: A, state$: Observable<S>, action$: Observable<A>): [
//     S,
//     Observable<A>
//   ];
// }

// const affector = (
//   state: State,
//   action: Action,
//   state$: Observable<State>,
//   action$: Observable<Action>
// ): [State, Observable<Action>] => {
//   switch (action.kind) {
//     case "LoginRequest":
//       return [
//         state,
//         of<Action>({ kind: "LoginSuccess", username: "poop" }).pipe(
//           delay(1500),
//           takeUntil(
//             action$.pipe(
//               filter(action => {
//                 switch (action.kind) {
//                   case "LoginRequest":
//                     return true;
//                 }
//                 return false;
//               })
//             )
//           )
//         )
//       ];
//   }
//   return [state, empty()];
// };

// const affect = (
//   init:State,
//   action$:Observable<Action>,
//   affector:(State, Action, Observable<State>, Observable<Action>): [
//   State,
//   Observable<Action>
// ]
// ): Observable<State> {

// }

// // const affect = (affector:AffectorFn<S,A>) {

// // }
// // const reactor<T, L> = (state:State, action, state$, action$):[]

// // what happens when multiple things subscribe to this observable?
// // should I be using a behavior subject inside this, or is there
// // a constructor for a behavior subject observable?
// const store$: Observable<State> = Observable.create(
//   (observer: Observer<State>) => {
//     const init: State = {};
//     const state$ = new BehaviorSubject<State>(init);
//     const action$ = action$$.pipe(flatMap(action$ => action$));
//     let last: State = init;
//     const sub = action$$.pipe(flatMap(action$ => action$)).subscribe(action => {
//       const next = reducer(last, action);
//       state$.next(next);
//     });
//     state$.subscribe(state => observer.next(state));

//     return () => {
//       sub.unsubscribe();
//       console.log("unsub");
//     };
//   }
// );

// BehaviorSubject.create;

// const action$$ = new Subject<Observable<Action>>();
// const createStore = (init: State) => {
//   const state$ = new BehaviorSubject<State>(init);
//   const state$ = action$$.pipe(
//     flatMap(action$ => action$),
//     scan(reduce, init),
//     tap()
//   );
// };

// const store = createStore({});

// store.subscribe(console.log);

// action$$.next(
//   of<Action>({ kind: "LoginRequest", username: "dingleberry", password: "pie" })
// );
// action$$.next(of<Action>({ kind: "logout" }).pipe(delay(1500)));
