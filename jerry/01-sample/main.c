#include <string.h>
#include "jerryscript.h"
#include "jerryscript-ext/handler.h"

int
main (void)
{
  const jerry_char_t script[] = "print ('Hello, World!');";
  size_t script_size = strlen ((const char *) script);

  /* Initialize engine */
  jerry_init (JERRY_INIT_EMPTY);

  /* Register 'print' function from the extensions */
  jerryx_handler_register_global ((const jerry_char_t *) "print",
                                  jerryx_handler_print);

  /* Setup Global scope code */
  jerry_value_t parsed_code = jerry_parse (script, script_size, false);

  if (!jerry_value_has_error_flag (parsed_code))
  {
    /* Execute the parsed source code in the Global scope */
    jerry_value_t ret_value = jerry_run (parsed_code);

    /* Returned value must be freed */
    jerry_release_value (ret_value);
  }

  /* Parsed source code must be freed */
  jerry_release_value (parsed_code);

  /* Cleanup engine */
  jerry_cleanup ();

  return 0;
}