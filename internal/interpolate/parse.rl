package interpolate

import "fmt"

%%{
    machine interpolate;
    write data;
}%%

// Parse parses a string for interpolation.
//
// Variables may be specified anywhere in the string in the format ${foo} or
// ${foo:default} where 'default' will be used if the variable foo was unset.
func Parse(data string) (out String, _ error) {
    var (
        // Variables used by Ragel
        cs  = 0         // current state
        p   = 0         // current position in data
        pe  = len(data)
        eof = pe        // eof == pe if this is the last data block

        // We use the following variables to actually build the String.

        // Index in data where the currently captured string started.
        idx int

        v variable  // variable being read, if any
        l literal   // literal being read, if any

        // Current term. This is either the variable that we just read or the
        // literal. We will append it to `out` and move on.
        t term
    )

    %%{
        # Record the current position as the start of a string. This is
        # usually used with the entry transition (>) to start capturing the
        # string when a state machine is entered.
        #
        # fpc is the current position in the string (basically the same as the
        # variable `p` but a special Ragel keyword) so after executing
        # `start`, data[idx:fpc+1] is the string from when start was called to
        # the current position (inclusive).
        action start { idx = fpc }

        # A variable always starts with an alphabet or an underscore and
        # contains alphanumeric characters, underscores, and non-consecutive
        # dots or dashes.
        var_name
            = ( [a-zA-Z_] ([a-zA-Z0-9_]
              | ('.' | '-') [a-zA-Z0-9_])*
              )
            >start
            @{ v.Name = data[idx:fpc+1] }
            ;

        var_default
            = (any - '}')* >start @{ v.Default = data[idx:fpc+1] };

        # var is a reference to a variable and optionally has a default value
        # for that variable.
        var = '${' var_name (':' @{ v.HasDefault = true } var_default)?  '}'
            ;

        # Anything followed by a '\' is used as-is.
        escaped_lit = '\\' any @{ l = literal(data[fpc:fpc+1]) };

        # Anything followed by a '$' that is not a '{' is used as-is with the
        # dollar.
        dollar_lit = '$' (any - '{') @{ l = literal(data[fpc-1:fpc+1]) };

        # Literal strings that don't contain '$' or '\'.
        simple_lit
            = (any - '$' - '\\')+
            >start
            @{ l = literal(data[idx:fpc + 1]) }
            ;

        lit = escaped_lit | dollar_lit | simple_lit;

        # Terms are the two possible components in a string. Either a literal
        # or a variable reference.
        term = (var @{ t = v }) | (lit @{ t = l });

        main := (term %{ out = append(out, t) })**;

        write init;
        write exec;
    }%%

    if cs < %%{ write first_final; }%% {
        return out, fmt.Errorf("cannot parse string %q", data)
    }

    return out, nil
}
