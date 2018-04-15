import { State } from "./state";
import { Action } from "./actions";

import { Observable, empty, of } from "rxjs";
import { delay, filter, takeUntil } from "rxjs/operators";

let count = 0;

export function affector(
  s: State,
  a: Action,
  s$: Observable<State>,
  a$: Observable<Action>
): [State, Observable<Action>] {
  if (count++ === 0) {
    s$.subscribe(state => console.log("state", state));
  }
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
          services: [
            { id: "2", name: "prod-nginx" },
            { id: "83", name: "prod-edge" }
          ]
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
    case "RefreshServicesSuccess": {
      let next: State = {
        ...s,
        service: { ...s.service, refreshing: false, services: a.services }
      };
      return [next, empty()];
    }
    case "SelectService":
      let next: State = {
        ...s,
        service: {
          ...s.service,
          selected: {
            service: a.service,
            refreshingColumns: true,
            columns: []
          }
        }
      };
      return [
        next,
        of<Action>({
          kind: "RefreshColumnsRequest",
          userID: s.user.id,
          serviceID: next.service.selected.service.id
        })
      ];
    case "RefreshColumnsRequest": {
      return [
        s,
        of<Action>({
          kind: "RefreshColumnsSuccess",
          columns: [
            {
              kind: "NumericColumn",
              field: "response_time"
            }
          ]
        })
      ];
    }
    case "RefreshColumnsSuccess": {
      let next: State = {
        ...s,
        service: {
          ...s.service,
          selected: {
            ...s.service.selected,
            refreshingColumns: false,
            calculate: undefined,
            columns: a.columns
          }
        }
      };
      return [next, empty()];
    }
    case "SelectCalculateColumn": {
      let next: State = {
        ...s,
        service: {
          ...s.service,
          selected: {
            ...s.service.selected,
            calculate: a.column
          }
        }
      };
      return [next, empty()];
    }
  }
  return [s, empty()];
}
