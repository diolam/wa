// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"wa-lang.org/wa/internal/ast"
	"wa-lang.org/wa/internal/constant"
	"wa-lang.org/wa/internal/token"
)

// A declInfo describes a package-level const, type, var, or func declaration.
type declInfo struct {
	file  *Scope        // scope of file containing this declaration
	lhs   []*Var        // lhs of n:1 variable declarations, or nil
	typ   ast.Expr      // type, or nil
	init  ast.Expr      // init/orig expression, or nil
	fdecl *ast.FuncDecl // func declaration, or nil
	alias bool          // type alias declaration

	// The deps field tracks initialization expression dependencies.
	deps objSet // lazily initialized
}

// An objSet is simply a set of objects.
type objSet map[Object]bool

// hasInitializer reports whether the declared object has an initialization
// expression or function body.
func (d *declInfo) hasInitializer() bool {
	return d.init != nil || d.fdecl != nil && d.fdecl.Body != nil
}

// addDep adds obj to the set of objects d's init expression depends on.
func (d *declInfo) addDep(obj Object) {
	m := d.deps
	if m == nil {
		m = make(objSet)
		d.deps = m
	}
	m[obj] = true
}

// arityMatch checks that the lhs and rhs of a const or var decl
// have the appropriate number of names and init exprs. For const
// decls, init is the value spec providing the init exprs; for
// var decls, init is nil (the init exprs are in s in this case).
func (check *Checker) arityMatch(s, init *ast.ValueSpec) {
	l := len(s.Names)
	r := len(s.Values)
	if init != nil {
		r = len(init.Values)
	}

	switch {
	case init == nil && r == 0:
		// var decl w/o init expr
		if s.Type == nil {
			check.errorf(s.Pos(), "missing type or init expr")
		}
	case l < r:
		if l < len(s.Values) {
			// init exprs from s
			n := s.Values[l]
			check.errorf(n.Pos(), "extra init expr %s", n)
			// TODO(gri) avoid declared but not used error here
		} else {
			// init exprs "inherited"
			check.errorf(s.Pos(), "extra init expr at %s", check.fset.Position(init.Pos()))
			// TODO(gri) avoid declared but not used error here
		}
	case l > r && (init != nil || r != 1):
		n := s.Names[r]
		check.errorf(n.Pos(), "missing init expr for %s", n)
	}
}

func validatedImportPath(path string) (string, error) {
	s, err := strconv.Unquote(path)
	if err != nil {
		return "", err
	}
	if s == "" {
		return "", fmt.Errorf("empty string")
	}
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	for _, r := range s {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
			return s, fmt.Errorf("invalid character %#U", r)
		}
	}
	return s, nil
}

// declarePkgObj declares obj in the package scope, records its ident -> obj mapping,
// and updates check.objMap. The object must not be a function or method.
func (check *Checker) declarePkgObj(ident *ast.Ident, obj Object, d *declInfo) {
	assert(ident.Name == obj.Name())

	// spec: "A package-scope or file-scope identifier with name init
	// may only be declared to be a function with this (func()) signature."
	if ident.Name == "init" {
		check.errorf(ident.Pos(), "cannot declare init - must be func")
		return
	}

	// spec: "The main package must have package name main and declare
	// a function main that takes no arguments and returns no value."
	if ident.Name == "main" && check.pkg.name == "main" {
		check.errorf(ident.Pos(), "cannot declare main - must be func")
		return
	}

	check.declare(check.pkg.scope, ident, obj, token.NoPos)
	check.objMap[obj] = d
	obj.setOrder(uint32(len(check.objMap)))
}

// filename returns a filename suitable for debugging output.
func (check *Checker) filename(fileNo int) string {
	file := check.files[fileNo]
	if pos := file.Pos(); pos.IsValid() {
		return check.fset.File(pos).Name()
	}
	return fmt.Sprintf("file[%d]", fileNo)
}

func (check *Checker) importPackage(pos token.Pos, path, dir string) *Package {
	// If we already have a package for the given (path, dir)
	// pair, use it instead of doing a full import.
	// Checker.impMap only caches packages that are marked Complete
	// or fake (dummy packages for failed imports). Incomplete but
	// non-fake packages do require an import to complete them.
	key := importKey{path, dir}
	imp := check.impMap[key]
	if imp != nil {
		return imp
	}

	// no package yet => import it
	if path == "C" && check.conf.FakeImportC {
		imp = NewPackage("C", "C")
		imp.fake = true
	} else {
		// ordinary import
		var err error
		if importer := check.conf.Importer; importer == nil {
			err = fmt.Errorf("Config.Importer not installed")
		} else if importerFrom, ok := importer.(ImporterFrom); ok {
			imp, err = importerFrom.ImportFrom(path, dir, 0)
			if imp == nil && err == nil {
				err = fmt.Errorf("Config.Importer.ImportFrom(%s, %s, 0) returned nil but no error", path, dir)
			}
		} else {
			imp, err = importer.Import(path)
			if imp == nil && err == nil {
				err = fmt.Errorf("Config.Importer.Import(%s) returned nil but no error", path)
			}
		}
		// make sure we have a valid package name
		// (errors here can only happen through manipulation of packages after creation)
		if err == nil && imp != nil && (imp.name == "_" || imp.name == "") {
			err = fmt.Errorf("invalid package name: %q", imp.name)
			imp = nil // create fake package below
		}
		if err != nil {
			check.errorf(pos, "could not import %s (%s)", path, err)
			if imp == nil {
				// create a new fake package
				// come up with a sensible package name (heuristic)
				name := path
				if i := len(name); i > 0 && name[i-1] == '/' {
					name = name[:i-1]
				}
				if i := strings.LastIndex(name, "/"); i >= 0 {
					name = name[i+1:]
				}
				imp = NewPackage(path, name)
			}
			// continue to use the package as best as we can
			imp.fake = true // avoid follow-up lookup failures
		}
	}

	// package should be complete or marked fake, but be cautious
	if imp.complete || imp.fake {
		check.impMap[key] = imp
		return imp
	}

	// something went wrong (importer may have returned incomplete package without error)
	return nil
}

// collectObjects collects all file and package objects and inserts them
// into their respective scopes. It also performs imports and associates
// methods with receiver base type names.
func (check *Checker) collectObjects() {
	pkg := check.pkg

	// pkgImports is the set of packages already imported by any package file seen
	// so far. Used to avoid duplicate entries in pkg.imports. Allocate and populate
	// it (pkg.imports may not be empty if we are checking test files incrementally).
	// Note that pkgImports is keyed by package (and thus package path), not by an
	// importKey value. Two different importKey values may map to the same package
	// which is why we cannot use the check.impMap here.
	var pkgImports = make(map[*Package]bool)
	for _, imp := range pkg.imports {
		pkgImports[imp] = true
	}

	var methods []*Func // list of methods with non-blank _ names
	for fileNo, file := range check.files {
		// The package identifier denotes the current package,
		// but there is no corresponding package object.
		check.recordDef(file.Name, nil)

		// Use the actual source file extent rather than *ast.File extent since the
		// latter doesn't include comments which appear at the start or end of the file.
		// Be conservative and use the *ast.File extent if we don't have a *token.File.
		pos, end := file.Pos(), file.End()
		if f := check.fset.File(file.Pos()); f != nil {
			pos, end = token.Pos(f.Base()), token.Pos(f.Base()+f.Size())
		}
		fileScope := NewScope(check.pkg.scope, pos, end, check.filename(fileNo))
		check.recordScope(file, fileScope)

		// determine file directory, necessary to resolve imports
		// FileName may be "" (typically for tests) in which case
		// we get "." as the directory which is what we would want.
		fileDir := dir(check.fset.Position(file.Name.Pos()).Filename)

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.BadDecl:
				// ignore

			case *ast.GenDecl:
				var last *ast.ValueSpec // last ValueSpec with type or init exprs seen
				for iota, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.ImportSpec:
						// import package
						path, err := validatedImportPath(s.Path.Value)
						if err != nil {
							check.errorf(s.Path.Pos(), "invalid import path (%s)", err)
							continue
						}

						imp := check.importPackage(s.Path.Pos(), path, fileDir)
						if imp == nil {
							continue
						}

						// add package to list of explicit imports
						// (this functionality is provided as a convenience
						// for clients; it is not needed for type-checking)
						if !pkgImports[imp] {
							pkgImports[imp] = true
							pkg.imports = append(pkg.imports, imp)
						}

						// local name overrides imported package name
						name := imp.name
						if s.Name != nil {
							name = s.Name.Name
							if path == "C" {
								// match cmd/compile (not prescribed by spec)
								check.errorf(s.Name.Pos(), `cannot rename import "C"`)
								continue
							}
							if name == "init" {
								check.errorf(s.Name.Pos(), "cannot declare init - must be func")
								continue
							}
						}

						obj := NewPkgName(s.Pos(), pkg, name, imp)
						obj.setNode(s)
						if s.Name != nil {
							// in a dot-import, the dot represents the package
							check.recordDef(s.Name, obj)
						} else {
							check.recordImplicit(s, obj)
						}

						if path == "C" {
							// match cmd/compile (not prescribed by spec)
							obj.used = true
						}

						// add import to file scope
						if name == "." {
							// merge imported scope with file scope
							for _, obj := range imp.scope.elems {
								// A package scope may contain non-exported objects,
								// do not import them!
								if obj.Exported() {
									// declare dot-imported object
									// (Do not use check.declare because it modifies the object
									// via Object.setScopePos, which leads to a race condition;
									// the object may be imported into more than one file scope
									// concurrently. See issue #32154.)
									if alt := fileScope.Insert(obj); alt != nil {
										check.errorf(s.Name.Pos(), "%s redeclared in this block", obj.Name())
										check.reportAltDecl(alt)
									}
								}
							}
							// add position to set of dot-import positions for this file
							// (this is only needed for "imported but not used" errors)
							check.addUnusedDotImport(fileScope, imp, s.Pos())
						} else {
							// declare imported package object in file scope
							// (no need to provide s.Name since we called check.recordDef earlier)
							check.declare(fileScope, nil, obj, token.NoPos)
						}

					case *ast.ValueSpec:
						switch d.Tok {
						case token.CONST:
							// determine which initialization expressions to use
							switch {
							case s.Type != nil || len(s.Values) > 0:
								last = s
							case last == nil:
								last = new(ast.ValueSpec) // make sure last exists
							}

							// declare all constants
							for i, name := range s.Names {
								obj := NewConst(name.Pos(), pkg, name.Name, nil, constant.MakeInt64(int64(iota)))
								obj.setNode(s)

								if s.Doc == nil {
									if d.Lparen == token.NoPos && d.Doc != nil {
										obj.setNodeDoc(d.Doc)
									}
								}

								var init ast.Expr
								if i < len(last.Values) {
									init = last.Values[i]
								}

								d := &declInfo{file: fileScope, typ: last.Type, init: init}
								check.declarePkgObj(name, obj, d)
							}

							check.arityMatch(s, last)

						case token.VAR, token.GLOBAL:
							lhs := make([]*Var, len(s.Names))
							// If there's exactly one rhs initializer, use
							// the same declInfo d1 for all lhs variables
							// so that each lhs variable depends on the same
							// rhs initializer (n:1 var declaration).
							var d1 *declInfo
							if len(s.Values) == 1 {
								// The lhs elements are only set up after the for loop below,
								// but that's ok because declareVar only collects the declInfo
								// for a later phase.
								d1 = &declInfo{file: fileScope, lhs: lhs, typ: s.Type, init: s.Values[0]}
							}

							// declare all variables
							for i, name := range s.Names {
								obj := NewVar(name.Pos(), pkg, name.Name, nil)
								obj.setNode(s)
								lhs[i] = obj

								if s.Doc == nil {
									if d.Lparen == token.NoPos && d.Doc != nil {
										obj.setNodeDoc(d.Doc)
									}
								}

								d := d1
								if d == nil {
									// individual assignments
									var init ast.Expr
									if i < len(s.Values) {
										init = s.Values[i]
									}
									d = &declInfo{file: fileScope, typ: s.Type, init: init}
								}

								check.declarePkgObj(name, obj, d)
							}

							check.arityMatch(s, nil)

						default:
							check.invalidAST(s.Pos(), "invalid token %s", d.Tok)
						}

					case *ast.TypeSpec:
						obj := NewTypeName(s.Name.Pos(), pkg, s.Name.Name, nil)
						obj.setNode(s)
						check.declarePkgObj(s.Name, obj, &declInfo{file: fileScope, typ: s.Type, alias: s.Assign.IsValid()})

						if s.Doc == nil {
							if d.Lparen == token.NoPos && d.Doc != nil {
								obj.setNodeDoc(d.Doc)
							}
						}

					default:
						check.invalidAST(s.Pos(), "unknown ast.Spec node %T", s)
					}
				}

			case *ast.FuncDecl:
				name := d.Name.Name
				obj := NewFunc(d.Name.Pos(), pkg, name, nil)
				obj.setNode(d)
				if d.Recv == nil {
					// regular function
					if name == "init" {
						// don't declare init functions in the package scope - they are invisible
						obj.parent = pkg.scope
						check.recordDef(d.Name, obj)
						// init functions must have a body
						if d.Body == nil {
							check.softErrorf(obj.pos, "missing function body")
						}
					} else {
						check.declare(pkg.scope, d.Name, obj, token.NoPos)
					}
				} else {
					// method
					// (Methods with blank _ names are never found; no need to collect
					// them for later type association. They will still be type-checked
					// with all the other functions.)
					if name != "_" {
						methods = append(methods, obj)
					}
					check.recordDef(d.Name, obj)
				}
				info := &declInfo{file: fileScope, fdecl: d}
				// Methods are not package-level objects but we still track them in the
				// object map so that we can handle them like regular functions (if the
				// receiver is invalid); also we need their fdecl info when associating
				// them with their receiver base type, below.
				check.objMap[obj] = info
				obj.setOrder(uint32(len(check.objMap)))

			default:
				check.invalidAST(d.Pos(), "unknown ast.Decl node %T", d)
			}
		}
	}

	// verify that objects in package and file scopes have different names
	for _, scope := range check.pkg.scope.children /* file scopes */ {
		for _, obj := range scope.elems {
			if alt := pkg.scope.Lookup(obj.Name()); alt != nil {
				if pkg, ok := obj.(*PkgName); ok {
					check.errorf(alt.Pos(), "%s already declared through import of %s", alt.Name(), pkg.Imported())
					check.reportAltDecl(pkg)
				} else {
					check.errorf(alt.Pos(), "%s already declared through dot-import of %s", alt.Name(), obj.Pkg())
					// TODO(gri) dot-imported objects don't have a position; reportAltDecl won't print anything
					check.reportAltDecl(obj)
				}
			}
		}
	}

	// Now that we have all package scope objects and all methods,
	// associate methods with receiver base type name where possible.
	// Ignore methods that have an invalid receiver. They will be
	// type-checked later, with regular functions.
	if methods == nil {
		return // nothing to do
	}
	check.methods = make(map[*TypeName][]*Func)
	for _, f := range methods {
		fdecl := check.objMap[f].fdecl
		if list := fdecl.Recv.List; len(list) > 0 {
			// f is a method.
			// Determine the receiver base type and associate f with it.
			ptr, base := check.resolveBaseTypeName(list[0].Type)
			if base != nil {
				f.hasPtrRecv = ptr
				check.methods[base] = append(check.methods[base], f)
			}
		}
	}
}

// resolveBaseTypeName returns the non-alias base type name for typ, and whether
// there was a pointer indirection to get to it. The base type name must be declared
// in package scope, and there can be at most one pointer indirection. If no such type
// name exists, the returned base is nil.
func (check *Checker) resolveBaseTypeName(typ ast.Expr) (ptr bool, base *TypeName) {
	// Algorithm: Starting from a type expression, which may be a name,
	// we follow that type through alias declarations until we reach a
	// non-alias type name. If we encounter anything but pointer types or
	// parentheses we're done. If we encounter more than one pointer type
	// we're done.
	var path []*TypeName
	for {
		typ = unparen(typ)

		// check if we have a pointer type
		if pexpr, _ := typ.(*ast.StarExpr); pexpr != nil {
			// if we've already seen a pointer, we're done
			if ptr {
				return false, nil
			}
			ptr = true
			typ = unparen(pexpr.X) // continue with pointer base type
		}

		// typ must be the name
		name, _ := typ.(*ast.Ident)
		if name == nil {
			return false, nil
		}

		// name must denote an object found in the current package scope
		// (note that dot-imported objects are not in the package scope!)
		obj := check.pkg.scope.Lookup(name.Name)
		if obj == nil {
			return false, nil
		}

		// the object must be a type name...
		tname, _ := obj.(*TypeName)
		if tname == nil {
			return false, nil
		}

		// ... which we have not seen before
		if check.cycle(tname, path, false) {
			return false, nil
		}

		// we're done if tdecl defined tname as a new type
		// (rather than an alias)
		tdecl := check.objMap[tname] // must exist for objects in package scope
		if !tdecl.alias {
			return ptr, tname
		}

		// otherwise, continue resolving
		typ = tdecl.typ
		path = append(path, tname)
	}
}

// cycle reports whether obj appears in path or not.
// If it does, and report is set, it also reports a cycle error.
func (check *Checker) cycle(obj *TypeName, path []*TypeName, report bool) bool {
	// (it's ok to iterate forward because each named type appears at most once in path)
	for i, prev := range path {
		if prev == obj {
			if report {
				check.errorf(obj.pos, "illegal cycle in declaration of %s", obj.name)
				// print cycle
				for _, obj := range path[i:] {
					check.errorf(obj.Pos(), "\t%s refers to", obj.Name()) // secondary error, \t indented
				}
				check.errorf(obj.Pos(), "\t%s", obj.Name())
			}
			return true
		}
	}
	return false
}

// packageObjects typechecks all package objects, but not function bodies.
func (check *Checker) packageObjects() {
	// process package objects in source order for reproducible results
	objList := make([]Object, len(check.objMap))
	i := 0
	for obj := range check.objMap {
		objList[i] = obj
		i++
	}
	sort.Sort(inSourceOrder(objList))

	// add new methods to already type-checked types (from a prior Checker.Files call)
	for _, obj := range objList {
		if obj, _ := obj.(*TypeName); obj != nil && obj.typ != nil {
			check.addMethodDecls(obj)
		}
	}

	// We process non-alias declarations first, in order to avoid situations where
	// the type of an alias declaration is needed before it is available. In general
	// this is still not enough, as it is possible to create sufficiently convoluted
	// recursive type definitions that will cause a type alias to be needed before it
	// is available (see issue #25838 for examples).
	// As an aside, the cmd/compiler suffers from the same problem (#25838).
	var aliasList []*TypeName
	// phase 1
	for _, obj := range objList {
		// If we have a type alias, collect it for the 2nd phase.
		if tname, _ := obj.(*TypeName); tname != nil && check.objMap[tname].alias {
			aliasList = append(aliasList, tname)
			continue
		}

		check.objDecl(obj, nil)
	}
	// phase 2
	for _, obj := range aliasList {
		check.objDecl(obj, nil)
	}

	// At this point we may have a non-empty check.methods map; this means that not all
	// entries were deleted at the end of typeDecl because the respective receiver base
	// types were not found. In that case, an error was reported when declaring those
	// methods. We can now safely discard this map.
	check.methods = nil
}

// inSourceOrder implements the sort.Sort interface.
type inSourceOrder []Object

func (a inSourceOrder) Len() int           { return len(a) }
func (a inSourceOrder) Less(i, j int) bool { return a[i].order() < a[j].order() }
func (a inSourceOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// processDelayed processes all delayed actions pushed after top.
func (check *Checker) processDelayed(top int) {
	for len(check.delayed) > top {
		i := len(check.delayed) - 1
		f := check.delayed[i]
		check.delayed = check.delayed[:i]
		f() // may append to check.delayed
	}
}

// unusedImports checks for unused imports.
func (check *Checker) unusedImports() {
	// if function bodies are not checked, packages' uses are likely missing - don't check
	if check.conf.IgnoreFuncBodies {
		return
	}

	// spec: "It is illegal (...) to directly import a package without referring to
	// any of its exported identifiers. To import a package solely for its side-effects
	// (initialization), use the blank identifier as explicit package name."

	// check use of regular imported packages
	for _, scope := range check.pkg.scope.children /* file scopes */ {
		for _, obj := range scope.elems {
			if obj, ok := obj.(*PkgName); ok {
				// Unused "blank imports" are automatically ignored
				// since _ identifiers are not entered into scopes.
				if !obj.used {
					path := obj.imported.path
					base := pkgName(path)
					if obj.name == base {
						check.softErrorf(obj.pos, "%q imported but not used", path)
					} else {
						check.softErrorf(obj.pos, "%q imported but not used as %s", path, obj.name)
					}
				}
			}
		}
	}

	// check use of dot-imported packages
	for _, unusedDotImports := range check.unusedDotImports {
		for pkg, pos := range unusedDotImports {
			check.softErrorf(pos, "%q imported but not used", pkg.path)
		}
	}
}

// pkgName returns the package name (last element) of an import path.
func pkgName(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		path = path[i+1:]
	}
	return path
}

// dir makes a good-faith attempt to return the directory
// portion of path. If path is empty, the result is ".".
// (Per the go/build package dependency tests, we cannot import
// path/filepath and simply use filepath.Dir.)
func dir(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i > 0 {
		return path[:i]
	}
	// i <= 0
	return "."
}
