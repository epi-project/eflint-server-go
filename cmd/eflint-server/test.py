import matplotlib.pyplot as plt
import matplotlib
import numpy as np
import readline

matplotlib.use("pgf")
matplotlib.rcParams.update({
    "pgf.texsystem": "pdflatex",
    'font.family': 'sans-serif',
    'font.size': 20,
    'text.usetex': True,
    'pgf.rcfonts': False,
    'savefig.transparent': True,
    'savefig.bbox': 'tight',
    'savefig.dpi': 300,
})

def format_bytes(x, pos):
    if x < 0:
        return ""

    for unit in ["bytes", "KB", "MB", "GB", "TB"]:
        if x < 1024.0:
            return f"{x:3.1f} {unit}"
        x /= 1024.0

def format_time(x, pos):
    if x < 0:
        return ""

    for unit in ["microseconds", "milliseconds"]:
        if x < 1000.0:
            return f"{x:3.1f} {unit}"
        x /= 1000.0


    return f"{x:3.1f} seconds"

def createDerivationTest():
    for i in range(5):
        x = 10 ** i
        with open("tests/performance/derivation/derivation_{}.eflint".format(x), "w") as f:
            f.writelines([f"Fact x Identified by 1..{x} Holds when x(x - 1).\n",
                          "+x(1).\n"])

def createDimensionalityTest():
    for i in range(1, 6):
        with open("tests/performance/dimensionality/dimensionality_{}.eflint".format(i), "w") as f:
            f.write("Fact parameter Identified by 1..10\n")
            f.write("Fact combined Identified by parameter1 ")
            for j in range(1, i):
                f.write(f"* parameter{j + 1} ")

            f.write(".\n")
            f.write("?-combined.\n")


def createCombinatorialTest():
    names = ["Dave", "Eve", "Frank", "George", "Helen"]

    for i in range(1, 6):
        with open("tests/performance/combinatorial/combinatorial_{}.eflint".format(i), "w") as f:
            f.write("Fact person Identified by Alice, Bob, Chloe")
            for name in names[:i]:
                f.write(", " + name)
            f.write(".\n")

            f.write("Fact family-of Identified by person1 * person2 Holds when family-of(person2, person1), family-of(person2, person3) && family-of(person3,person1).\n")
            f.write("+family-of(Alice, Bob).\n")
            f.write("+family-of(Bob, Chloe).\n")


def plotDerivationTest():
    # Execution time in nanoseconds
    haskell = [159516487, 156278781, 199409027, 19541309248, 0]
    golang  = [332295,    2902475,   29607432,  367736588,   8190766005]
    #          399948,    3188478,   52012894,  397365187,   9341365596
    x = [10 ** i for i in range(5)]
    index = np.arange(len(x))

    plt.figure(figsize=(10, 5))
    width = 0.3

    plt.bar(index, haskell, width, label="Reference interpreter")
    plt.bar(index + width, golang, width, label="Our work")

    plt.yscale("log")

    plt.xlabel("Domain Size")
    plt.ylabel("Time (seconds)")

    plt.xticks(index + width / 2, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_time))

    plt.legend(loc="best")
    plt.tight_layout()
    plt.savefig("figures/derivation_execution.pgf", format="pgf", dpi=300)

    # Memory usage in bytes
    haskell = [15944, 15944,  16176,   44616,    0]
    golang  = [76600, 613130, 5819520, 57978168, 586865528]

    plt.figure(figsize=(10, 5))
    plt.bar(index, haskell, width, label="Reference interpreter")
    plt.bar(index + width, golang, width, label="Our work")

    plt.yscale("log")

    plt.xlabel("Domain Size")
    plt.ylabel("Memory (bytes)")

    plt.xticks(index + width / 2, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_bytes))

    plt.legend(loc="upper left")
    plt.tight_layout()
    plt.savefig("figures/derivation_memory.pgf", format="pgf", dpi=300)


def plotDimensionalityTest():
    # Execution time in nanoseconds
    haskell = [145924372, 145694949, 158399191, 203652844, 690475732]
    golang  = [322937,    2588482,   32433689,  427137076, 5884419860]

    x = [10 ** i for i in range(5)]
    index = np.arange(len(x))

    plt.figure(figsize=(10, 5))
    width = 0.3

    plt.bar(index, haskell, width, label="Reference interpreter")
    plt.bar(index + width, golang, width, label="Our work")

    plt.yscale("log")

    plt.xlabel("Parameter Size")
    plt.ylabel("Time (seconds)")

    plt.xticks(index + width / 2, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_time))

    plt.legend(loc="best")
    plt.tight_layout()
    plt.savefig("figures/dimensionality_execution.pgf", format="pgf", dpi=300)

    # Memory usage in bytes
    haskell = [15944, 28232,  143336,   2109002,   16789064]
    golang  = [89360, 674813, 10359241, 126654637, 1832955776]

    plt.figure(figsize=(10, 5))
    plt.bar(index, haskell, width, label="Reference interpreter")
    plt.bar(index + width, golang, width, label="Our work")

    plt.yscale("log")

    plt.xlabel("Parameter Size")
    plt.ylabel("Memory (bytes)")

    plt.xticks(index + width / 2, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_bytes))

    plt.legend(loc="upper left")
    plt.tight_layout()
    plt.savefig("figures/dimensionality_memory.pgf", format="pgf", dpi=300)


def plotCombinatorialTest():
    # Execution time in nanoseconds
    golang = [74333057, 132607159, 264913470, 329927800, 519327296]

    x = [10 ** i for i in range(5)]
    index = np.arange(len(x))

    plt.figure(figsize=(10, 5))
    width = 0.3
    plt.bar(index, golang, width, label="Our work", color="#ff7f0e")

    plt.yscale("log")

    plt.xlabel("Domain Size")
    plt.ylabel("Time (seconds)")

    plt.xticks(index, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_time))

    plt.legend(loc="best")
    plt.tight_layout()
    plt.savefig("figures/combinatorial_execution.pgf", format="pgf", dpi=300)

    # Memory usage in bytes
    golang = [14452264, 24243337, 38365774, 57173794, 81905288]

    plt.figure(figsize=(10, 5))
    plt.bar(index, golang, width, label="Our work", color="#ff7f0e")

    plt.yscale("log")

    plt.xlabel("Domain Size")
    plt.ylabel("Memory (bytes)")

    plt.xticks(index, x)
    plt.gca().yaxis.set_major_formatter(matplotlib.ticker.FuncFormatter(format_bytes))

    plt.legend(loc="upper left")
    plt.tight_layout()
    plt.savefig("figures/cozmbinatorial_memory.pgf", format="pgf", dpi=300)


def main():
    # createDerivationTest()
    plotDerivationTest()

    # createDimensionalityTest()
    plotDimensionalityTest()

    # createCombinatorialTest()
    plotCombinatorialTest()

if __name__ == "__main__":
    main()
