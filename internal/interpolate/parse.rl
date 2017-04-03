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
        // Ragel variables
        cs  = 0
        p   = 0
        pe  = len(data)
        eof = pe

        // The following variables are used by us to build String up.

        // Index in data where the currently captured string started.
        idx   int

        // Variable currently being built.
        v     variable

        // Literal currently being read.
        l     literal

        // Last read term (variable or literal) which we will append to the
        // output.
        t     term
    )

    %%{
        # Record the current position as the start of a string.
        action start { idx = fpc }

        var_name
            = ([a-zA-Z_] ([a-zA-Z0-9_] | '.' [a-zA-Z0-9_])*)
            >start
            @{ v.Name = data[idx:fpc+1] };

        var_default
            = (any - '}')* >start @{ v.Default = data[idx:fpc+1] };

        var = '${' var_name (':' @{ v.HasDefault = true } var_default)? '}';

        lit = ('\\' any @{ l = literal(data[fpc:fpc+1]) })
            | ((any - [\$\\])+ >start @{ l = literal(data[idx:fpc + 1]) })
            ;

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
