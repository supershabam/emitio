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
      return [
        { ...s, user: { name: a.name, id: a.id } },
        of<Action>({ kind: "RefreshServicesRequest", userID: a.id })
      ];
    case "Logout":
      return [{ ...s, user: null }, empty()];
    case "RefreshServicesRequest":
      return [
        { ...s, service: { ...s.service, refreshing: true } },
        of<Action>({
          kind: "RefreshServicesSuccess",
          services: ["prod-nginx", "prod-edge"]
        }).pipe(
          delay(400),
          takeUntil(
            a$.pipe(
              filter(a => {
                return a.kind === "RefreshServicesRequest";
              })
            )
          )
        )
      ];
    case "RefreshServicesSuccess":
      return [
        {
          ...s,
          service: { ...s.service, refreshing: false, services: a.services }
        },
        empty()
      ];
    case "SelectService":
      return [
        { ...s, service: { ...s.service, selected: a.selected } },
        empty()
      ];
  }
  return [s, empty()];
}
