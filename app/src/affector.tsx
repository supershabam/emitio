import { State } from "./state";
import { Action } from "./actions";

import { Observable, empty, of } from "rxjs";
import { delay, filter, takeUntil } from "rxjs/operators";

export function affector(
  s: State,
  a: Action,
  s$: Observable<State>,
  a$: Observable<Action>
): [State, Observable<Action>] {
  console.log("affector", s, a);
  switch (a.kind) {
    case "LoginRequest":
      return [
        s,
        of<Action>({
          kind: "LoginSuccess",
          name: a.username,
          id: "12345"
        }).pipe(
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
      return [{ ...s, user: { name: a.name, id: a.id } }, empty()];
    case "Logout":
      return [{ ...s, user: null }, empty()];
    case "SelectService":
      return [
        { ...s, service: { ...s.service, selected: a.selected } },
        empty()
      ];
  }
  return [s, empty()];
}
