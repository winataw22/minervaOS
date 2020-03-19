from yaml import load, CLoader
import json
import sys
import os


def main(args):
    if len(args) == 0:
        raise Exception("required file name")
    name = args[0]
    input = open(name)

    data = load(input, CLoader)
    input.close()

    output_name = "%s.json" % name
    output = open(output_name, 'w')
    try:
        json.dump(data, output)
    except Exception as e:
        os.remove(output_name)
        raise e
    finally:
        output.close()


if __name__ == '__main__':
    main(sys.argv[1:])
